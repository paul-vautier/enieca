package plugin_test

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"
)

// EnergyMiddleWareConfig the plugin configuration.
type EnergyMiddleWareConfig struct {
	urlGreenEnergy string
	configuration [3]RequestConfigurations
	duration int
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *EnergyMiddleWareConfig {
	return &EnergyMiddleWareConfig{
		"",
		[3]RequestConfigurations{
			{},
			{},
			{},
		},
		60,
	}
}

type EnergyMiddleware struct {
	next           http.Handler
	name           string
	mu             *sync.Mutex
	configurations [3]map[string]ParametersValues
}

// New created a new plugin.

func New(ctx context.Context, next http.Handler, config *EnergyMiddleWareConfig, name string) (http.Handler, error) {
	scheduler := Scheduler{
		config.configuration,
	}

	middleware := &EnergyMiddleware{
		next,
		name,
		new(sync.Mutex),
		scheduler.GetNextConfiguration(0, 0, 0, 0, CONF_HIGH, config.duration),
	}

	go middleware.runBackgroundTask(ctx, scheduler, config.duration)

	return middleware, nil
}

func (e *EnergyMiddleware) runBackgroundTask(ctx context.Context, scheduler Scheduler, duration int) {
	// Use a ticker to run the background task periodically
	ticker := time.NewTicker(time.Second * time.Duration(duration))
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			// TODO : appel HTTP pour récupérer l'énergie verte
			os.Stdout.WriteString("running background task")
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
