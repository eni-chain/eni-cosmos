package tasks

import "github.com/prometheus/client_golang/prometheus"

// Prometheus' metrics definitions
var (
	executeTxDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "execute_tx_duration_seconds",
			Help:    "Duration of transaction execution in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		},
	)
	processAllDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "process_all_duration_seconds",
			Help:    "Total duration of ProcessAll in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 15), // 10ms to ~160s
		},
	)
	findConflictsDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "find_conflicts_duration_seconds",
			Help:    "Duration of conflict detection in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15), // 100µs to ~1.6s
		},
	)
	parallelExecDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "parallel_exec_duration_seconds",
			Help:    "Duration of parallel task execution in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 15), // 10ms to ~160s
		},
	)
	taskStatusTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_status_total",
			Help: "Total number of tasks by status",
		},
		[]string{"status"},
	)
)

// Register metrics with Prometheus
func init() {
	prometheus.MustRegister(executeTxDuration)
	prometheus.MustRegister(processAllDuration)
	prometheus.MustRegister(findConflictsDuration)
	prometheus.MustRegister(parallelExecDuration)
	prometheus.MustRegister(taskStatusTotal)
}
