package worker

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
	// worker "github.com/weldpua2008/supraworker/worker"
	"github.com/weldpua2008/supraworker/job"
	"github.com/weldpua2008/supraworker/model"
)

var (
	log = logrus.WithFields(logrus.Fields{"package": "worker"})
)

// StartWorker run goroutine for executing commands and reporting to your API
// Note that a WaitGroup must be passed to functions by
// pointer.
func StartWorker(id int, jobs <-chan *model.Job, wg *sync.WaitGroup) {
	// On return, notify the WaitGroup that we're done.
	defer func() {
		wg.Done()
		log.Debug(fmt.Sprintf("Worker %v finished ", id))
	}()

	log.Info(fmt.Sprintf("Starting worker %v", id))
	for j := range jobs {
		log.Trace(fmt.Sprintf("Worker %v received Job %v address %p", id, j.Id, &j))
		if err := j.Run(); err != nil {
			log.Info(fmt.Sprintf("Job %v failed with %s", j.Id, err))
			if errFlushBuf := j.FlushSteamsBuffer(); errFlushBuf != nil {
				log.Tracef("Job %v failed to flush buffer due %v", j.Id, errFlushBuf)
			}
			_ = j.Failed()
		} else {
			dur := time.Since(j.StartAt)
			log.Debugf("Job %v finished in %v", j.Id, dur)
			if errFlushBuf := j.FlushSteamsBuffer(); errFlushBuf != nil {
				log.Tracef("Job %v failed to flush buffer due %v", j.Id, errFlushBuf)
			}
			_ = j.Finish()
		}
		job.JobsRegistry.Delete(j.StoreKey())

	}
}
