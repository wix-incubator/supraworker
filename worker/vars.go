package worker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	log           = logrus.WithFields(logrus.Fields{"package": "worker"})
	jobsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "supraworker_processed_jobs_total",
		Help: "The total number of processed jobs",
	})
	jobsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "supraworker_jobs_failed_total",
		Help: "The total number of FAILED jobs",
	})
	jobsSucceeded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "supraworker_jobs_succeeded_total",
		Help: "The total number of SUCCEEDED jobs",
	})

	jobsDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "supraworker_jobs_duration_secs",
		Help:    "The Jobs duration in seconds.",
		Buckets: prometheus.LinearBuckets(20, 5, 5),
	})
)
