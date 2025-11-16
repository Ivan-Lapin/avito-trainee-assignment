package metrics

import (
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	AssignmentsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "assignments_total",
		Help: "Total number of initial reviewer assignments",
	})
	ReassignmentsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reassignments_total",
		Help: "Total number of reassign operations",
	})
	MergesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "merges_total",
		Help: "Total number of merges",
	})
	RateLimitedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rate_limited_total",
		Help: "Total number of 429 responses",
	})
	IdempotencyHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "idempotency_hits_total",
		Help: "Total idempotency cache hits",
	})
	RequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_request_duration_ms",
		Help:    "HTTP request duration in milliseconds",
		Buckets: []float64{10, 25, 50, 100, 200, 300, 500, 1000},
	})
)

// Register регистрирует /metrics и готов к расширению для middleware измерений.
func Register(r *chi.Mux) {
	r.Handle("/metrics", promhttp.Handler())
}
