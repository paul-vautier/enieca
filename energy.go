package enieca

const (
	CONF_LOW = iota 
	CONF_MEDIUM
	CONF_HIGH
)

type Scheduler struct {
	configurations map[string]RequestConfigurations
}
type RequestConfigurations struct {
	qoe               []int
	median_joules_req []float64
	parameters        []BenchmarkParameters
}

func (s *Scheduler) GetNextConfiguration(
	req_rate_sust int,
	req_rate_bal int,
	req_rate_perf int,
	available_green float64 ,
	duration_seconds int) map[string][3]SelectedConfiguration{

	endpointsParam := make(map[string][3]SelectedConfiguration) 

	for endpoint, conf := range s.configurations {

		conf_min := argMin(conf.qoe)
		conf_max := argMax(conf.qoe)

		conf_perf := conf_max
		conf_sust := conf_min
		conf_bal := conf_min

		if conf.power(conf_sust, req_rate_sust) + conf.power(conf_min, req_rate_bal) < available_green {
			conf_bal = conf.optimize(req_rate_bal, conf_sust, req_rate_sust, available_green)
			conf_sust = conf.optimize(req_rate_sust, conf_bal, req_rate_bal, available_green)

		}
		endpointsParam[endpoint] = [3]SelectedConfiguration{
			{conf.parameters[conf_sust], conf.median_joules_req[conf_perf] - conf.median_joules_req[conf_sust]},
			{conf.parameters[conf_bal], conf.median_joules_req[conf_perf] - conf.median_joules_req[conf_bal]},
			{conf.parameters[conf_perf], conf.median_joules_req[conf_perf] - conf.median_joules_req[conf_perf]},
		}
	}
	return endpointsParam
}

// Calculates the  consumption of a specific configuration under a current load type
func (r *RequestConfigurations) power(conf int, req_rate int) float64 {
       return r.median_joules_req[conf] * float64(req_rate)
}

// Calculates the total energy that a request will consume under a specific load type for a duration in seconds 
func (r *RequestConfigurations) energy(conf int, req_rate int, duration int) float64 {
       return r.power(conf, req_rate) * float64(duration)
}

// Returns the index of the most adapted configuration given the current green power, load and request rate
func (r *RequestConfigurations) optimize(requests_rate int, conf_subtract int, requests_rate_subtract int, power_green float64) int {

	var conf_candidates_indices []int 
	for i := 0; i < len(r.qoe); i++ {
		if r.power(i, requests_rate) <= (power_green - r.power(conf_subtract, requests_rate_subtract)) {
			conf_candidates_indices = append(conf_candidates_indices, i)
		}
	}

	conf_max_qoe_valid := argMax(r.qoe, conf_candidates_indices...)
	max_qoe := r.qoe[conf_max_qoe_valid]
	min_power_conf_idx := conf_max_qoe_valid
	min_power := r.power(conf_max_qoe_valid, requests_rate)

	for _, current_candidate_idx := range conf_candidates_indices {
		if r.qoe[current_candidate_idx] == max_qoe {
			current := r.power(current_candidate_idx, requests_rate)
			if current < min_power {
				min_power = current
				min_power_conf_idx = current_candidate_idx
			}	
		}
	}

	return min_power_conf_idx
}
