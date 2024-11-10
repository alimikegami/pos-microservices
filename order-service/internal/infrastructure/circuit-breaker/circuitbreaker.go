package circuitbreaker

import "github.com/sony/gobreaker/v2"

func CreateCircuitBreaker(name string) *gobreaker.CircuitBreaker[[]byte] {
	var st gobreaker.Settings
	st.Name = name
	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.Requests >= 3 && failureRatio >= 0.6
	}

	cb := gobreaker.NewCircuitBreaker[[]byte](st)

	return cb
}
