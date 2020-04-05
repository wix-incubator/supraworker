package model

import (
	"context"
	"fmt"
	"time"
)

func StartHeartBeat(ctx context.Context, interval time.Duration) error {

	chanSentHeartBeats := make(chan int, 1)
	chanFailedToSentHeartBeats := make(chan int, 1)

	go func() {
		tickerSendHeartBeats := time.NewTicker(interval)
		defer func() {
			tickerSendHeartBeats.Stop()
		}()

		hb_all := 0
		hb_failed := 0
		for {
			select {
			case <-ctx.Done():
				chanSentHeartBeats <- hb_all
				chanFailedToSentHeartBeats <- hb_failed
				log.Debug("Heartbeat generation finished [ SUCESSFULLY ]")
				return
			case <-tickerSendHeartBeats.C:
				stage := "heartbeat.update"
				if urlProvided(stage) {
					params := GetAPIParamsFromSection(stage)
					if errApi, result := DoApiCall(ctx, params, stage); errApi != nil {
						log.Tracef("failed to update api, got: %s and %s\n", result, errApi)
						hb_failed += 1
					}
					hb_all += 1
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
