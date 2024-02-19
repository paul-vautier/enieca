package plugin_test

import "fmt"

const (
	CONF_LOW = iota 
	CONF_MEDIUM
	CONF_HIGH
)

type Scheduler struct {
	configurations [3]RequestConfigurations
}
type ParametersValues struct {
	name []string
	value []string
}
func (p *ParametersValues) FindByName(name string) (string, error) {
	for i, v := range p.name {
		if v == name {
			return p.value[i], nil
		}
	}
	return "", fmt.Errorf("value %s not found in the array", name)
}
type RequestConfigurations struct {
	qoe               []int
	max_joules_req    []float64
	min_joules_req    []float64
	mean_joules_req   []float64
	median_joules_req []float64
	parameters        []map[string]ParametersValues
}

func (s *Scheduler) GetNextConfiguration(
	req_rate_sust int,
	req_rate_bal int,
	req_rate_perf int,
	available_green float64 ,
	load_type int,
	duration_seconds int) [3]map[string]ParametersValues {

	conf_min := argMin(s.configurations[load_type].qoe)
	conf_max := argMax(s.configurations[load_type].qoe)

	conf_perf := conf_max
	conf_sust := conf_min
	conf_bal := conf_min
	if s.power(load_type, conf_sust, req_rate_sust) + s.power(load_type, conf_min, req_rate_bal) < available_green {
		conf_bal = s.optimize(req_rate_bal, conf_sust, req_rate_sust, load_type, available_green)
		conf_sust = s.optimize(req_rate_sust, conf_bal, req_rate_bal, load_type, available_green)
	}
	return [3]map[string]ParametersValues{
		s.configurations[load_type].parameters[conf_sust],
		s.configurations[load_type].parameters[conf_bal],
		s.configurations[load_type].parameters[conf_perf],
	}
}

// Calculates the  consumption of a specific configuration under a current load type
func (s *Scheduler) power(load int, conf int, req_rate int) float64 {
       return s.configurations[load].median_joules_req[conf] * float64(req_rate)
}

// Calculates the total energy that a request will consume under a specific load type for a duration in seconds 
func (s *Scheduler) energy(load int, conf int, req_rate int, duration int) float64 {
       return s.power(load, conf, req_rate) * float64(duration)
}

// Returns the index of the most adapted configuration given the current green power, load and request rate
func (s *Scheduler) optimize(load int, requests_rate int, conf_subtract int, requests_rate_subtract int, power_green float64) int {

	var conf_candidates_indices []int 

	for i := 0; i < len(s.configurations[load].qoe); i++ {
		if s.power(load, i, requests_rate) <= (power_green - s.power(load, conf_subtract, requests_rate_subtract)) {
			conf_candidates_indices = append(conf_candidates_indices, i)
		}
	}

	conf_max_qoe_valid := argMax(s.configurations[load].qoe, conf_candidates_indices...)
	max_qoe := s.configurations[load].qoe[conf_max_qoe_valid]
	min_power_conf_idx := conf_max_qoe_valid
	min_power := s.power(load, conf_max_qoe_valid, requests_rate)

	for _, current_candidate_idx := range conf_candidates_indices {
		if s.configurations[load].qoe[current_candidate_idx] == max_qoe {
			current := s.power(load, current_candidate_idx, requests_rate)
			if current < min_power {
				min_power = current
				min_power_conf_idx = current_candidate_idx
			}	
		}
	}

	return min_power_conf_idx
}
