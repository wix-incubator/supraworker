package metrics

import (
	// "sync"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	FetchCancelLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "supraworker",
		Subsystem: "jobs_cancel",
		Name:      "latency_ns",
		Help:      "The latency distribution of jobs in cancellation flow processed",
	},
		[]string{"type", "namespace", "service"},
	)
	FetchNewJobLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "supraworker",
		Subsystem: "jobs_fetch",
		Name:      "latency_ns",
		Help:      "The latency distribution of new jobs processed",
	},
		[]string{"type", "namespace", "service"},
	)
	WorkerStatistics = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "supraworker",
			Subsystem: "worker",
			Name:      "stats",
			Help:      "Statistics of workers.",
		},
		[]string{
			// Type
			"type",
			// What is the Operation?
			"operation",
			"namespace",
			"service",
		},
	)

	JobsFetchDuplicates = promauto.NewCounter(prometheus.CounterOpts{
		Name: "supraworker_fetch_jobs_duplicates_total",
		Help: "The total number of fetched duplicated jobs",
	})
	JobsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "supraworker_fetch_jobs_total",
		Help: "The total number of fetched jobs",
	})
	JobsCancelled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "supraworker_jobs_cancelled_total",
		Help: "The total number of CANCELLED jobs",
	})
)
