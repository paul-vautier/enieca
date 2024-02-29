package enieca

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type BenchmarkParameters []BenchmarkParameter

func (p BenchmarkParameters) FindByName(name string) (string, error) {
	for _, v := range p {
		if v.Name == name {
			return v.Value, nil
		}
	}
	return "", fmt.Errorf("value %s not found in the array", name)
}

type TypeParameters []TypeParameter

func (p TypeParameters) FindByName(name string) (string, error) {
	for _, v := range p {
		if v.Name == name {
			return v.Type, nil
		}
	}
	return "", fmt.Errorf("value %s not found in the array", name)
}

// EnergyMiddlewareConfig the plugin configuration.
type EnergyMiddlewareConfig struct {
	UrlGreenEnergy     string     `yaml:"url_green_energy"`
	DefaultGreenEnergy float64    `yaml:"default_green_energy"`
	Duration           int        `yaml:"duration"`
	Endpoints          []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Name       string          `yaml:"name"`
	Redirect   string          `yaml:"redirect"`
	Parameters []TypeParameter `yaml:"parameters"`
	Benchmark  []Benchmark     `yaml:"benchmark"`
}

type TypeParameter struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type Benchmark struct {
	QOE                    int                 `yaml:"qoe"`
	MeanRPS                float64             `yaml:"mean_rps"`
	MedianRPS              int                 `yaml:"median_rps"`
	MinRPS                 int                 `yaml:"min_rps"`
	MaxRPS                 int                 `yaml:"max_rps"`
	MeanJoulesPerRequest   float64             `yaml:"mean_joules_per_request"`
	MedianJoulesPerRequest float64             `yaml:"median_joules_per_request"`
	MinJoulesPerRequest    float64             `yaml:"min_joules_per_request"`
	MaxJoulesPerRequest    float64             `yaml:"max_joules_per_request"`
	Parameters             BenchmarkParameters `yaml:"parameters"`
}

// For each endpoint, create a map endpoint_name > request configurations
func endpointsToConfigMap(endpoints []Endpoint) map[string]RequestConfigurations {

	requestPerEndpoints := make(map[string]RequestConfigurations)

	for _, endpoint := range endpoints {
		endpointConfig := RequestConfigurations{}
		for _, benchmark := range endpoint.Benchmark {
			endpointConfig.median_joules_req = append(endpointConfig.median_joules_req, benchmark.MedianJoulesPerRequest)
			endpointConfig.qoe = append(endpointConfig.qoe, benchmark.QOE)
			endpointConfig.parameters = append(endpointConfig.parameters, benchmark.Parameters)
		}
		requestPerEndpoints[endpoint.Name] = endpointConfig
	}

	return requestPerEndpoints
}

func endpointToRedirections(endpoints []Endpoint) (map[string]TypeParameters, map[string]string) {

	parametersType := make(map[string]TypeParameters)
	redirections := make(map[string]string)

	for _, endpoint := range endpoints {
		parametersType[endpoint.Name] = endpoint.Parameters
		redirections[endpoint.Name] = endpoint.Redirect
	}

	return parametersType, redirections
}

type BenchmarkParameter struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *EnergyMiddlewareConfig {
	return &EnergyMiddlewareConfig{}
}

type SelectedConfiguration struct {
	parameters          BenchmarkParameters
	savedEnergy         float64
	expectedConsumption float64
}
type EnergyMiddleware struct {
	next               http.Handler
	name               string
	mu                 *sync.RWMutex
	configurations     map[string][3]SelectedConfiguration
	urlGreenEnergy     string
	defaultGreenEnergy float64
	parametersType     map[string]TypeParameters
	redirections       map[string]string
	req_sust           atomic.Uint32
	req_bal            atomic.Uint32
	req_perf           atomic.Uint32
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *EnergyMiddlewareConfig, name string) (http.Handler, error) {

	scheduler := Scheduler{
		endpointsToConfigMap(config.Endpoints),
	}
	duration := config.Duration
	power_green := 0
	params, redirections := endpointToRedirections(config.Endpoints)

	middleware := &EnergyMiddleware{
		next,
		name,
		new(sync.RWMutex),
		scheduler.GetNextConfiguration(0, 0, 0, float64(power_green)),
		config.UrlGreenEnergy,
		config.DefaultGreenEnergy,
		params,
		redirections,
		atomic.Uint32{},
		atomic.Uint32{},
		atomic.Uint32{},
	}

	go middleware.runBackgroundTask(ctx, scheduler, duration)
	return middleware, nil
}

func (e *EnergyMiddleware) runBackgroundTask(ctx context.Context, scheduler Scheduler, duration int) {
	ticker := time.NewTicker(time.Second * time.Duration(duration))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			power_green := e.findAvailableGreenEnergy()
			e.mu.Lock()
			e.configurations = scheduler.GetNextConfiguration(int(e.req_sust.Swap(0)), int(e.req_bal.Swap(0)), int(e.req_perf.Swap(0)), power_green * float64(duration))
			e.mu.Unlock()
			fmt.Fprintf(os.Stdout, "[INFO] Expected power consumption in watts for the next %d seconds : \n", duration)
			for k, v := range e.configurations {
				fmt.Fprintf(os.Stdout, "[INFO] Endpoint %s\n", k)
				fmt.Fprintf(os.Stdout, "[INFO] Sustained %f (W)\n", v[0].expectedConsumption)
				fmt.Fprintf(os.Stdout, "[INFO] Balanced %f (W)\n", v[1].expectedConsumption)
				fmt.Fprintf(os.Stdout, "[INFO] Performance %f (W)\n", v[2].expectedConsumption)
			}
		}
	}
}

func (e *EnergyMiddleware) findAvailableGreenEnergy() float64 {
	if e.urlGreenEnergy != "" {
		resp, err := http.Get(e.urlGreenEnergy)
		if err != nil {
			fmt.Fprintf(os.Stdout, "[ERROR] Request to %s failed\n", e.urlGreenEnergy)
			return e.defaultGreenEnergy
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			fmt.Fprintf(os.Stdout, "[ERROR] Request to %s returned a non-2xx status code\n", e.urlGreenEnergy)
			return e.defaultGreenEnergy
		}

		// Unmarshal the JSON
		body, err := io.ReadAll(resp.Body)
		// Define a struct to match the JSON structure
		// Assuming `data` is now of type map[string]interface{}
		var data map[string]interface{}

		// Unmarshal the JSON
		err = json.Unmarshal(body, &data)
		if err != nil {
		    fmt.Fprintf(os.Stdout, "Error unmarshalling body %s : %s\n", string(body), err)
		    return e.defaultGreenEnergy
		}

		// Access the desired value
		results, ok := data["results"].([]interface{})
		if ok && len(results) > 0 {
		    series, ok := results[0].(map[string]interface{})["series"].([]interface{})
		    if ok && len(series) > 0 {
			values, ok := series[0].(map[string]interface{})["values"].([]interface{})
			if ok && len(values) > 0 {
			    if len(values[0].([]interface{})) >= 2 {
				value, ok := values[0].([]interface{})[1].(float64)
				if ok {
				    fmt.Fprintf(os.Stdout, "[INFO] The available green energy is %f\n", value)
				    return value
				}
			    }
			}
		    }
		}
		fmt.Fprintf(os.Stdout, "[ERROR] invalid json body for %s\n", e.urlGreenEnergy)
	}
	return e.defaultGreenEnergy
}

func (e *EnergyMiddleware) registerRequestType(req http.Request) int {
	header_value := req.Header.Get("X-user-energy-objective")
	switch header_value {
	case "eco":
		e.req_sust.Add(1)
		return CONF_LOW
	case "balanced":
		e.req_bal.Add(1)
		return CONF_MEDIUM
	case "high":
		fallthrough
	default:
		e.req_perf.Add(1)
		return CONF_HIGH
	}
}

func (e *EnergyMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	e.mu.RLock()
	newReq := *req
	endpoint := newReq.URL.Path
	confForEndpoint, exists := e.configurations[endpoint]

	if exists {
		conf := confForEndpoint[e.registerRequestType(newReq)]
		url := newReq.URL
		redirectUrl := e.redirections[endpoint]
		parameters := e.parametersType[endpoint]

		for _, param := range parameters {
			paramVal, err := conf.parameters.FindByName(param.Name)
			if err != nil {
				fmt.Fprintf(os.Stdout, "[ERROR] Could not find a parameter with name %s for request %s\n", param.Name, url.Path)
				continue
			}
			switch param.Type {
			case "path":
				redirectUrl = strings.Replace(redirectUrl, "{:"+param.Name+"}", paramVal, -1)
			case "query":
				queries := newReq.URL.Query()
				queries.Del(param.Name)
				queries.Add(param.Name, paramVal)
				newReq.URL.RawQuery = queries.Encode()
			default:
				continue
			}
		}
		newReq.URL.Path = redirectUrl
		newReq.RequestURI = newReq.URL.RequestURI()
		rw.Header().Add("X-energy-economy", fmt.Sprintf("%f", conf.savedEnergy))
	} 

	e.mu.RUnlock()
	e.next.ServeHTTP(rw, &newReq)

}
