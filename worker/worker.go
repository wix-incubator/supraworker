package worker

import (
	"errors"
	"fmt"
	"github.com/weldpua2008/supraworker/config"
	"github.com/weldpua2008/supraworker/job"
	"github.com/weldpua2008/supraworker/metrics"
	"github.com/weldpua2008/supraworker/model"
	"github.com/weldpua2008/supraworker/utils"
	"sync"
	"sync/atomic"
	"time"
)

// StartWorker run goroutine for executing commands and reporting to your API
// Note that a WaitGroup must be passed to functions by
// pointer.
// There are several scenarios for the Job execution:
//	1). Job execution finished with error/success [Regular flow]
//	2). Cancelled because of TTR [Timeout]
//	3). Cancelled by Job's Registry because of Cleanup process (TTR) [Cancel]
//	4). Cancelled when we fetch external API (cancellation information) [Cancel]

func StartWorker(id int, jobs <-chan *model.Job, wg *sync.WaitGroup) {
	workerId:=fmt.Sprintf("worker-%d", id)
	logWorker := log.WithField("worker", workerId)
	// On return, notify the WaitGroup that we're done.
	defer func() {
		logWorker.Debugf("[FINISHED]")
		metrics.WorkerStatistics.WithLabelValues(
			"finished", workerId, config.C.PrometheusNamespace, config.C.PrometheusService,
		).Inc()
		wg.Done()
	}()

	logWorker.Info("Starting")
	metrics.WorkerStatistics.WithLabelValues(
		"live", workerId, config.C.PrometheusNamespace, config.C.PrometheusService,
	).Inc()
	for j := range jobs {
		j.AddToContext(utils.CtxWorkerIdKey, workerId)
		logJob := j.GetLogger()

		logJob.Tracef("New Job with TTR %v", time.Duration(j.TTR)*time.Millisecond)
		metrics.WorkerStatistics.WithLabelValues(
			"new_job", workerId, config.C.PrometheusNamespace, config.C.PrometheusService,
		).Inc()
		atomic.AddInt64(&NumActiveJobs, 1)

		errJobRun := j.Run()
		if errFlushBuf := j.FlushSteamsBuffer(); errFlushBuf != nil {
			logJob.Tracef("failed to flush logstream buffer due %v", errFlushBuf)
		}

		dur := time.Since(j.StartAt)
		switch {
		// Execution stopped by TTR
		case errors.Is(errJobRun, model.ErrJobTimeout):
			if errTimeout := j.Timeout(); errTimeout != nil {
				logJob.Tracef("[Timeout()] got: %v ", errTimeout)
			}
			metrics.JobsTimeout.Inc()
		case errors.Is(errJobRun, model.ErrJobCancelled):
			if errTimeout := j.Cancel(); errTimeout != nil {
				logJob.Tracef("[Cancel()] got: %v ", errTimeout)
			}
			metrics.JobsTimeout.Inc()
		case errJobRun == nil:
			if err := j.Finish(); err != nil {
				logJob.Debugf("finished in %v got %v", dur, err)
			} else {
				logJob.Debugf("finished in %v", dur)
			}
			metrics.JobsSucceeded.Inc()
			metrics.JobsDuration.Observe(dur.Seconds())
		default:
			if errFail := j.Failed(); errFail != nil {
				logJob.Tracef("[Failed()] got: %v ", errFail)
			}
			metrics.JobsFailed.Inc()
			logJob.Infof("Failed with %s", errJobRun)
		}

		job.JobsRegistry.Delete(j.StoreKey())
		atomic.AddInt64(&NumActiveJobs, -1)
		atomic.AddInt64(&NumProcessedJobs, 1)

		metrics.JobsProcessed.Inc()
		metrics.WorkerStatistics.WithLabelValues(
			"processed_job", workerId, config.C.PrometheusNamespace, config.C.PrometheusService,
		).Inc()

	}
}
