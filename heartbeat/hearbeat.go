package heartbeat

import (
	"context"
	"fmt"
	communicator "github.com/weldpua2008/supraworker/communicator"
	// config "github.com/weldpua2008/supraworker/config"
	// utils "github.com/weldpua2008/supraworker/config"
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

				// param := utils.ConvertMapStringToInterface(
				// 	config.GetStringMapStringTemplated(section, config.CFG_PREFIX_COMMUNICATORS))
				param := make(map[string]interface{}, 0)
				clusterCtx, cancel := context.WithTimeout(ctx, time.Duration(5)*time.Second)
				// log.Tracef("param %v" , param)
				defer cancel() // cancel when we are getting the kill signal or exit

				for _, comm := range comms {
					comm.Configure(param)
					// if err:=comm.Configure(param);err != nil {
					//     log.Tracef("comm.Configure %v => %v", comm, err)
					//
					// }
					// log.Tracef( "param %v in comm", param, comm)
					res, err := comm.Fetch(clusterCtx, param)
					if err != nil {
						log.Tracef("Can't send healthcheck %v got %v", err, res)
						continue
					}
				}
				// stage := "heartbeat.update"
				// if urlProvided(stage) {
				// 	params := GetAPIParamsFromSection(stage)
				// 	if errApi, result := DoApiCall(ctx, params, stage); errApi != nil {
				// 		log.Tracef("failed to update api, got: %s and %s\n", result, errApi)
				// 		hbFailed += 1
				// 	}
				// 	hbAll += 1
				// }
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
