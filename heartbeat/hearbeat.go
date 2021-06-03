package heartbeat

import (
	"context"
	"fmt"
	"github.com/weldpua2008/supraworker/communicator"
	"github.com/weldpua2008/supraworker/config"
	"github.com/weldpua2008/supraworker/worker"
	"sync/atomic"
	"time"
)

// StartHeartBeat which should update your API.
func StartHeartBeat(ctx context.Context, section string, interval time.Duration) error {

	chanSentHeartBeats := make(chan int, 1)
	chanFailedToSentHeartBeats := make(chan int, 1)
	communicators, err := communicator.GetCommunicatorsFromSection(section)
	if err != nil {
		comm, err1 := communicator.GetSectionCommunicator(section)
		if err1 == nil {
			communicators = []communicator.Communicator{comm}
		}
	}
	param := make(map[string]interface{})

	if err != nil {
		return fmt.Errorf("%w", communicator.ErrNoSuitableCommunicator)
	}
	log.Infof("Starting heartbeat with delay %v", interval)

	go func() {
		tickerHeartBeats := time.NewTicker(interval)

		defer func() {
			log.Tracef("Stopping hearbeat ticker")
			tickerHeartBeats.Stop()
		}()

		hbAll := 0
		hbFailed := 0
		for {
			select {
			case <-ctx.Done():
				chanSentHeartBeats <- hbAll
				chanFailedToSentHeartBeats <- hbFailed
				log.Debug("Heartbeat generation finished [ SUCCESSFULLY ]")
				return
			case <-tickerHeartBeats.C:

				func() {
					heartbeatCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
					// TODO: wrap it in a function –either an anonymous or a named function
					defer cancel() // cancel when we are getting the kill signal or exit
					config.C.NumActiveJobs = worker.NumActiveJobs
					config.C.NumFreeSlots = int(int64(config.C.NumWorkers) - atomic.LoadInt64(&config.C.NumActiveJobs))
					for _, comm := range communicators {
						_ = comm.Configure(param)
						res, err := comm.Fetch(heartbeatCtx, param)
						if err != nil {
							log.Tracef("Can't send healthcheck %v got %v", err, res)
							hbFailed += 1
							continue
						}
						hbAll += 1
					}
				}()
			}
		}
	}()
	numSentHB := <-chanSentHeartBeats
	numFailedHB := <-chanFailedToSentHeartBeats

	if numFailedHB > 0 {
		log.Infof("Number of failed heatbeats %v of all hertbeats  %v\n", numFailedHB, numSentHB)
	} else {
		log.Debugf("Number of sent heatbeats %v\n", numSentHB)
	}
	return nil

}
