package enieca

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
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

type Parameters []Parameter

func (p Parameters) FindByName(name string) (string, error) {
	for _, v := range p {
		if v.Name == name {
			return v.Type, nil
		}
	}
	return "", fmt.Errorf("value %s not found in the array", name)
}

// EnergyMiddleWareConfig the plugin configuration.
type EnergyMiddleWareConfig struct {
	UrlGreenEnergy string `yaml:"url_green_energy"`

	Duration  int        `yaml:"duration"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Name       string      `yaml:"name"`
	Redirect   string      `yaml:"redirect"`
	Parameters []Parameter `yaml:"parameters"`
	Benchmark  []Benchmark `yaml:"benchmark"`
}

type Parameter struct {
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

func endpointToRedirections(endpoints []Endpoint) (map[string]Parameters, map[string]string) {

	parametersType := make(map[string]Parameters)
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
func CreateConfig() *EnergyMiddleWareConfig {
	// Populate the struct with the provided
	/*
		medium_conf := RequestConfigurations{}
		parameters := []ParametersValues{
			{[]string{"size", "filter"}, []string{"32", "0"}},
			{[]string{"size", "filter"}, []string{"32", "1"}},
			{[]string{"size", "filter"}, []string{"32", "2"}},
			{[]string{"size", "filter"}, []string{"32", "3"}},
			{[]string{"size", "filter"}, []string{"128", "0"}},
			{[]string{"size", "filter"}, []string{"128", "1"}},
			{[]string{"size", "filter"}, []string{"128", "2"}},
			{[]string{"size", "filter"}, []string{"128", "3"}},
			{[]string{"size", "filter"}, []string{"512", "0"}},
			{[]string{"size", "filter"}, []string{"512", "1"}},
			{[]string{"size", "filter"}, []string{"512", "2"}},
			{[]string{"size", "filter"}, []string{"512", "3"}},
			{[]string{"size", "filter"}, []string{"1024", "0"}},
			{[]string{"size", "filter"}, []string{"1024", "1"}},
			{[]string{"size", "filter"}, []string{"1024", "2"}},
			{[]string{"size", "filter"}, []string{"1024", "3"}},
		}

		medium_conf.parameters = parameters
			medium_conf.mean_joules_req = []float64{0.017966, 0.018308, 0.018680, 0.018395, 0.023815, 0.028352, 0.031419, 0.035495, 0.080277, 0.140719, 0.193998, 0.265554, 0.087588, 0.195831, 0.261377, 0.357143}
			medium_conf.min_joules_req = []float64{0.017300, 0.017522, 0.018057, 0.017628, 0.022979, 0.027153, 0.030169, 0.034379, 0.075029, 0.131068, 0.185132, 0.253952, 0.081671, 0.181523, 0.248314, 0.338360}
			medium_conf.max_joules_req = []float64{0.018657, 0.019323, 0.019355, 0.019224, 0.024715, 0.029578, 0.032984, 0.037228, 0.086668, 0.150326, 0.205957, 0.276696, 0.094285, 0.211726, 0.274446, 0.374594}
		medium_conf.median_joules_req = []float64{0.017978, 0.018263, 0.018711, 0.018272, 0.023732, 0.028180, 0.030914, 0.034846, 0.077757, 0.140628, 0.189009, 0.266462, 0.087326, 0.195145, 0.261324, 0.356625}
		medium_conf.qoe = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	*/
	return &EnergyMiddleWareConfig{}
}

type SelectedConfiguration struct {
	parameters  BenchmarkParameters
	savedEnergy float64
}
type EnergyMiddleware struct {
	next           http.Handler
	name           string
	mu             *sync.Mutex
	configurations map[string][3]SelectedConfiguration
	urlGreenEnergy string
	parametersType map[string]Parameters
	redirections   map[string]string
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *EnergyMiddleWareConfig, name string) (http.Handler, error) {

	fmt.Fprintf(os.Stdout, "%+v\n", config)
	scheduler := Scheduler{
		endpointsToConfigMap(config.Endpoints),
	}
	duration := config.Duration
	power_green := 0
	req_rate_sust := 250
	req_rate_bal := 150
	req_rate_perf := 120
	params, redirections := endpointToRedirections(config.Endpoints)

	middleware := &EnergyMiddleware{
		next,
		name,
		new(sync.Mutex),
		scheduler.GetNextConfiguration(req_rate_sust, req_rate_bal, req_rate_perf, float64(power_green), duration),
		"",
		params,
		redirections,
	}

	go middleware.runBackgroundTask(ctx, scheduler, duration)
	return middleware, nil
}

func (e *EnergyMiddleware) runBackgroundTask(ctx context.Context, scheduler Scheduler, duration int) {
	// Use a ticker to run the background task periodically
	ticker := time.NewTicker(time.Second * time.Duration(duration))
	power_green := 20.
	req_rate_sust := 250
	req_rate_bal := 150
	req_rate_perf := 120
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			power_green = e.findAvailableGreenEnergy()
			e.mu.Lock()
			e.configurations = scheduler.GetNextConfiguration(req_rate_sust, req_rate_bal, req_rate_perf, power_green, duration)
			fmt.Fprintf(os.Stdout, "%+v\n", e.configurations)
			fmt.Fprintf(os.Stdout, "%+v\n", e.configurations)
			fmt.Fprintf(os.Stdout, "%+v\n", e.configurations)
			e.mu.Unlock()
		}
	}
}

func (e *EnergyMiddleware) findAvailableGreenEnergy() float64 {
	if e.urlGreenEnergy != "" {

	}
	return 0
}

func getEnergyHeaderValue(req http.Request) int {
	header_value := req.Header.Get("X-user-energy-objective")
	switch header_value {
	case "eco":
		return CONF_LOW
	case "balanced":
		return CONF_MEDIUM
	case "high":
		return CONF_HIGH
	default:
		return CONF_HIGH
	}
}

func (e *EnergyMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	e.mu.Lock()
	newReq := *req
	endpoint := newReq.URL.Path
	confForEndpoint, exists := e.configurations[endpoint]

	if exists {
		conf := confForEndpoint[getEnergyHeaderValue(newReq)]
		url := newReq.URL
		redirectUrl := e.redirections[endpoint]
		parameters := e.parametersType[endpoint]

		for _, param := range parameters {
			paramVal, err := conf.parameters.FindByName(param.Name)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Could not find a parameter with name %s for request %s", param.Name, url.Path)
				break
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
				break
			}
		}
		newReq.URL.Path = redirectUrl
		fmt.Fprintf(os.Stdout, "%+v\n", newReq.URL.Path)
		newReq.RequestURI = newReq.URL.RequestURI()
		newReq.Header.Add("X-energy-economy", fmt.Sprintf("%f", conf.savedEnergy))
		e.mu.Unlock()
	}

	e.next.ServeHTTP(rw, &newReq)
}
