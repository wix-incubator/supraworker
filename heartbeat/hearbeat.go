package heartbeat

import (
	"context"
	"fmt"
	"github.com/weldpua2008/supraworker/communicator"
	"github.com/weldpua2008/supraworker/config"
	"github.com/weldpua2008/supraworker/worker"
	"time"
)

// StartHeartBeat which should update your API.
func StartHeartBeat(ctx context.Context, section string, interval time.Duration) error {

	chanSentHeartBeats := make(chan int, 1)
	chanFailedToSentHeartBeats := make(chan int, 1)
	comms, err := communicator.GetCommunicatorsFromSection(section)
	if err != nil {
		comm, err1 := communicator.GetSectionCommunicator(section)
		if err1 == nil {
			comms := make([]communicator.Communicator, 0)
			comms = append(comms, comm)
		}
	}
	param := make(map[string]interface{})

	if err != nil {
		return fmt.Errorf("%w", communicator.ErrNoSuitableCommunicator)
	}
	log.Info(fmt.Sprintf("Starting heartbeat with delay %v", interval))

	go func() {
		tickerSendHeartBeats := time.NewTicker(interval)

		defer func() {
			tickerSendHeartBeats.Stop()
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
			case <-tickerSendHeartBeats.C:

				clusterCtx, cancel := context.WithTimeout(ctx, time.Duration(15)*time.Second)
				defer cancel() // cancel when we are getting the kill signal or exit
				config.C.NumActiveJobs = worker.NumActiveJobs
				config.C.NumFreeSlots = config.C.NumWorkers - config.C.NumActiveJobs
				for _, comm := range comms {
					_ = comm.Configure(param)
					res, err := comm.Fetch(clusterCtx, param)
					if err != nil {
						log.Tracef("Can't send healthcheck %v got %v", err, res)
						hbFailed += 1
						continue
					}
					hbAll += 1
				}
			}
		}
	}()
	numSentHB := <-chanSentHeartBeats
	numFailedHB := <-chanFailedToSentHeartBeats

	if numFailedHB > 0 {
		log.Info(fmt.Sprintf("Number of failed heatbeats %v of all hertbeats  %v\n", numFailedHB, numSentHB))
	} else {
		log.Debug(fmt.Sprintf("Number of sent heatbeats %v\n", numSentHB))
	}
	return nil

}
