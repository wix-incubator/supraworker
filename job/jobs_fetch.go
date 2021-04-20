package job

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	model "github.com/weldpua2008/supraworker/model"
	"strconv"
	"strings"
	"time"
)

var (
	log = logrus.WithFields(logrus.Fields{"package": "job"})
	// Registry for the Jobs
	JobsRegistry = model.NewRegistry()
)

// ApiJobRequest is struct for new jobs
type ApiJobRequest struct {
	JobStatus string `json:"job_status"`
	Limit     int64  `json:"limit"`
}

// An ApiJobResponse represents a Job response.
// Example response
// {
//   "job_id": "dbd618f0-a878-e477-7234-2ef24cb85ef6",
//   "jobStatus": "RUNNING",
//   "has_error": false,
//   "error_msg": "",
//   "run_uid": "0f37a129-eb52-96a7-198b-44515220547e",
//   "job_name": "Untitled",
//   "cmd": "su  - hadoop -c 'hdfs ls ''",
//   "parameters": [],
//   "createDate": "1583414512",
//   "lastUpdated": "1583415483",
//   "stopDate": "1586092912",
//   "extra_run_id": "scheduled__2020-03-05T09:21:40.961391+00:00"
// }
type ApiJobResponse struct {
	JobId       string   `json:"job_id"`
	JobStatus   string   `json:"jobStatus"`
	JobName     string   `json:"job_name"`
	RunUID      string   `json:"run_uid"`
	ExtraRunUID string   `json:"extra_run_id"`
	CMD         string   `json:"cmd"`
	Parameters  []string `json:"parameters"`
	CreateDate  string   `json:"createDate"`
	LastUpdated string   `json:"lastUpdated"`
	StopDate    string   `json:"stopDate"`
	EnvVar      []string `json:"env"`
}

// NewApiJobRequest prepare struct for Jobs for execution request
func NewApiJobRequest() *ApiJobRequest {
	return &ApiJobRequest{
		JobStatus: "PENDING",
		Limit:     5,
	}
}

// StartGenerateJobs goroutine for getting jobs from API with internal
// it expects `model.FetchNewJobAPIURL`
// exists on kill
func StartGenerateJobs(ctx context.Context, jobs chan *model.Job, interval time.Duration) error {
	if len(model.FetchNewJobAPIURL) < 1 {
		close(jobs)
		log.Warn("Please provide URL to fetch new Jobs")
		return fmt.Errorf("FetchNewJobAPIURL is undefined")
	}
	doneNumJobs := make(chan int, 1)
	doneNumCancelJobs := make(chan int, 1)
	log.Info(fmt.Sprintf("Starting generate jobs with delay %v", interval))
	tickerCancelJobs := time.NewTicker(10 * time.Second)
	tickerGenerateJobs := time.NewTicker(interval)
	defer func() {
		tickerGenerateJobs.Stop()
		tickerCancelJobs.Stop()
	}()

	go func() {
		j := 0
		for {
			select {
			case <-ctx.Done():
				close(jobs)
				doneNumJobs <- j
				if GracefullShutdown(jobs) {
					log.Debug("Jobs generation finished [ SUCCESSFULLY ]")
				} else {
					log.Warn("Jobs generation finished [ FAILED ]")
				}

				return
			case <-tickerGenerateJobs.C:
				if err, jobsData := model.NewRemoteApiRequest(ctx, "jobs.get.params", model.FetchNewJobAPIMethod, model.FetchNewJobAPIURL); err == nil {

					for _, jobResponse := range jobsData {
						var JobId string
						var CMD string
						var RunUID string
						var ExtraRunUID string
						var TTR uint64
						var EnvVar []string

						for key, value := range jobResponse {
							switch strings.ToLower(key) {
							case "id", "jobid", "job_id", "job_uid":
								JobId = fmt.Sprintf("%v", value)
							case "cmd", "command", "execute":
								CMD = fmt.Sprintf("%v", value)
							case "ttl", "ttr", "ttl_sec", "ttl_seconds":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									TTR = uint64((time.Duration(i) * time.Second).Milliseconds())
								}
							case "ttl_minutes", "ttr_minutes", "ttl_min", "ttr_min":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									TTR = uint64((time.Duration(i) * time.Minute).Milliseconds())
								}
							case "stopDate", "stopdate", "stop_date", "stop":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									now := time.Now()
									sec := now.Unix() // number of seconds since January 1, 1970 UTC
									if (i - sec) > 0 {
										TTR = uint64((time.Duration(i-sec) * time.Second).Milliseconds())
									}

								}
							case "runid", "runuid", "run_id", "run_uid":
								RunUID = fmt.Sprintf("%v", value)
							// Excpects list of string or string in key=value format
							case "env", "vars", "environment":
								if env, ok := value.([]string); ok {
									EnvVar = env
								} else if env, ok := value.(string); ok {
									EnvVar = []string{env}
								} else if value_of_slice, ok := value.([]interface{}); ok {
									for _, elem := range value_of_slice {
										if env, ok := elem.(string); ok {
											EnvVar = append(EnvVar, env)
										}
									}

								}

							case "extrarunid", "extrarunuid", "extrarun_id", "extrarun_uid", "extra_run_id", "extra_run_uid":
								ExtraRunUID = fmt.Sprintf("%v", value)
							}
						}
						if len(JobId) < 1 {
							continue
						}

						job := model.NewJob(fmt.Sprintf("%v", JobId), CMD)
						job.RunUID = RunUID
						job.ExtraRunUID = ExtraRunUID
						job.RawParams = append(job.RawParams, jobResponse)
						job.SetContext(ctx)
						if TTR < 1 {

							TTR = uint64((time.Duration(8*3600) * time.Second).Milliseconds())
						}
						job.TTR = TTR
						if JobsRegistry.Add(job) {
							jobsProcessed.Inc()
							jobs <- job
							j += 1
							log.Trace(fmt.Sprintf("sent job id %v ", job.Id))
						}
						job.CmdENV = EnvVar
						// else {
						// 	log.Trace(fmt.Sprintf("Duplicated job id %v ", job.Id))
						// }
					}
				} else {
					log.Trace(fmt.Sprintf("Failed fetch a new Jobs portion due %v ", err))
				}

			}
		}
	}()

	// Single goroutine for canceling jobs
	// We are getting such jobs from API
	// exists on kill

	log.Info(fmt.Sprintf("Starting canceling jobs with delay %v", interval))

	go func() {
		j := 0
		for {
			select {
			case <-ctx.Done():
				doneNumCancelJobs <- j
				log.Debug("Jobs cancelation finished [ SUCCESSFULLY ]")

				return
			case <-tickerCancelJobs.C:

				n := JobsRegistry.Cleanup()
				if n > 0 {
					j += n
					log.Trace(fmt.Sprintf("Cleared %v/%v jobs", n, j))
				}

				stage := "jobs.cancelation"
				params := model.GetAPIParamsFromSection(stage)
				if err, jobsCancelationData := model.DoApiCall(ctx, params, stage); err != nil {
					log.Tracef("failed to update api, got: %s and %s", jobsCancelationData, err)
				} else {

					for _, jobResponse := range jobsCancelationData {
						var JobId string
						var RunUID string
						var ExtraRunUID string

						for key, value := range jobResponse {
							switch strings.ToLower(key) {
							case "id", "jobid", "job_id", "job_uid":
								JobId = fmt.Sprintf("%v", value)
							case "runid", "runuid", "run_id", "run_uid":
								RunUID = fmt.Sprintf("%v", value)
							case "extrarunid", "extrarunuid", "extrarun_id", "extrarun_uid", "extra_run_id", "extra_run_uid":
								ExtraRunUID = fmt.Sprintf("%v", value)
							}
						}
						if len(JobId) < 1 {
							continue
						}
						jobCancelationId := model.StoreKey(JobId, RunUID, ExtraRunUID)
						if jobCancelation, ok := JobsRegistry.Record(jobCancelationId); ok {
							if err := jobCancelation.Cancel(); err != nil {
								log.Tracef("Can't cancel '%s' got %s ", jobCancelationId, err)
							}
						}
					}
				}
			}

		}

	}()

	numSentJobs := <-doneNumJobs
	numCancelJobs := <-doneNumCancelJobs

	log.Info(fmt.Sprintf("Sent %v jobs", numSentJobs))
	if numCancelJobs > 0 {
		log.Info(fmt.Sprintf("Canceled %v jobs", numCancelJobs))
	}
	return nil
}

// GracefullShutdown cancel all running jobs
// returns error in case any job failed to cancel
func GracefullShutdown(jobs <-chan *model.Job) bool {
	// empty jobs channel
	if len(jobs) > 0 {
		log.Trace(fmt.Sprintf("jobs chan still has size %v, empty it", len(jobs)))
		for len(jobs) > 0 {
			<-jobs
		}
	}
	JobsRegistry.GracefullyShutdown()
	if JobsRegistry.Len() > 0 {
		log.Trace(fmt.Sprintf("GracefullyShutdown failed, '%v' jobs left ", JobsRegistry.Len()))
		return false
	}
	return true

}
