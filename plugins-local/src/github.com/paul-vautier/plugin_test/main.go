package plugin_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// EnergyMiddleWareConfig the plugin configuration.
type EnergyMiddleWareConfig struct {
	urlGreenEnergy string
	configuration  [3]RequestConfigurations
	duration       int
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *EnergyMiddleWareConfig {
	medium_conf := RequestConfigurations{}
	// Populate the struct with the provided
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
	return &EnergyMiddleWareConfig{
		"",
		[3]RequestConfigurations{
			{},
			medium_conf,
			{},
		},
		60,
	}
}

type EnergyMiddleware struct {
	next           http.Handler
	name           string
	mu             *sync.Mutex
	configurations [3]ParametersValues
}

// New created a new plugin.

func New(ctx context.Context, next http.Handler, config *EnergyMiddleWareConfig, name string) (http.Handler, error) {
	scheduler := Scheduler{
		config.configuration,
	}
	duration := 60
	power_green := 20
	req_rate_sust := 250
	req_rate_bal := 150
	req_rate_perf := 120

	middleware := &EnergyMiddleware{
		next,
		name,
		new(sync.Mutex),
		scheduler.GetNextConfiguration(req_rate_sust, req_rate_bal, req_rate_perf, float64(power_green), CONF_MEDIUM, duration),
	}

	fmt.Fprintf(os.Stdout, "%+v\n", middleware.configurations[0])
	fmt.Fprintf(os.Stdout, "%+v\n", middleware.configurations[1])
	fmt.Fprintf(os.Stdout, "%+v\n", middleware.configurations[2])
	go middleware.runBackgroundTask(ctx, scheduler, duration)

	return middleware, nil
}

func (e *EnergyMiddleware) runBackgroundTask(ctx context.Context, scheduler Scheduler, duration int) {
	// Use a ticker to run the background task periodically
	ticker := time.NewTicker(time.Second * time.Duration(duration))
	power_green := 20
	req_rate_sust := 250
	req_rate_bal := 150
	req_rate_perf := 120
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			e.configurations = scheduler.GetNextConfiguration(req_rate_sust, req_rate_bal, req_rate_perf, float64(power_green), CONF_MEDIUM, duration)
			e.mu.Unlock()
		}
	}
}

func (e *EnergyMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	e.mu.Lock()
	newReq := *req
	newReq.URL = req.URL.JoinPath("/3")
	newReq.RequestURI = newReq.URL.RequestURI()
	e.mu.Unlock()
	e.next.ServeHTTP(rw, &newReq)
}
