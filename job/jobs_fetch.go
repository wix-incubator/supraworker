package job

import (
	"context"
	"fmt"
	"github.com/weldpua2008/supraworker/config"
	"github.com/weldpua2008/supraworker/metrics"
	"github.com/weldpua2008/supraworker/model"
	"strconv"
	"strings"
	"time"
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

// StartGenerateJobs goroutine for getting jobs from API with internal
// it expects `model.FetchNewJobAPIURL`
// exists on kill
func StartGenerateJobs(ctx context.Context, jobs chan *model.Job, interval time.Duration, maxRequestTimeout time.Duration) error {
	if len(model.FetchNewJobAPIURL) < 1 {
		close(jobs)
		log.Warn("Please provide URL to fetch new Jobs")
		return fmt.Errorf("FetchNewJobAPIURL is undefined")
	}
	doneNumJobs := make(chan int, 1)
	doneNumCancelJobs := make(chan int, 1)
	log.Infof("Starting generate jobs with delay %v", interval)
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
				if GracefulShutdown(jobs) {
					log.Debug("Jobs generation finished [ SUCCESSFULLY ]")
				} else {
					log.Warn("Jobs generation finished [ FAILED ]")
				}

				return
			case <-tickerGenerateJobs.C:
				start := time.Now()
				// Cleanup all processed jobs
				// we do this in this thread since new jobs can include jobs that are already on workers
				if n := JobsRegistry.Cleanup(); n > 0 {
					j += n
					log.Tracef("Cleared %d/%d, already processed %d jobs", n, n+JobsRegistry.Len(), j)
					//JobsRegistry.Map(func(key string, job *model.Job) {
					//	log.Tracef("Left Job %s => %p in %s cmd: %s", job.StoreKey(), job, job.Status, job.CMD)
					//})
				}
				// TODO: customize timeout
				if err, jobsData := model.NewRemoteApiRequest(context.WithValue(ctx, model.CtxKeyRequestTimeout, maxRequestTimeout), "jobs.get.params", model.FetchNewJobAPIMethod, model.FetchNewJobAPIURL); err == nil {
					metrics.FetchNewJobLatency.WithLabelValues(
						"api_get", config.C.PrometheusNamespace, config.C.PrometheusService,
					).Observe(float64(time.Since(start).Nanoseconds()))
					for _, jobResponse := range jobsData {
						var JobId string
						var CMD string
						var RunUID string
						var ExtraRunUID string
						var TTR uint64
						var EnvVar []string
						metrics.FetchNewJobLatency.WithLabelValues(
							"new_job_response", config.C.PrometheusNamespace, config.C.PrometheusService,
						).Observe(float64(time.Since(start).Nanoseconds()))

						for key, value := range jobResponse {

							switch strings.ToLower(key) {
							case "id", "jobid", "job_id", "job_uid":
								JobId = fmt.Sprintf("%v", value)
							case "cmd", "command", "execute":
								CMD = fmt.Sprintf("%v", value)
							case "ttl_minutes", "ttr_minutes", "ttl_min", "ttr_min":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									TTR = uint64((time.Duration(i) * time.Minute).Milliseconds())
								}
							case "ttl", "ttr", "ttl_sec", "ttl_seconds":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									TTR = uint64((time.Duration(i) * time.Second).Milliseconds())
								}
							case "ttl_msec", "ttr_msec":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									TTR = uint64((time.Duration(i) * time.Millisecond).Milliseconds())
								}
							case "stopDate", "stopdate", "stop_date", "stop":
								if i, err := strconv.ParseInt(fmt.Sprintf("%v", value), 10, 64); err == nil {
									sec := time.Now().Unix() // number of seconds since January 1, 1970 UTC
									if (i - sec) > 0 {
										TTR = uint64((time.Duration(i-sec) * time.Second).Milliseconds())
									}
								}
							case "runid", "runuid", "run_id", "run_uid":
								RunUID = fmt.Sprintf("%v", value)
							// Expects one of the following:
							//	- list of string
							//	- string in key=value format
							case "env", "vars", "environment":
								if env, ok := value.([]string); ok {
									EnvVar = env
								} else if env, ok := value.(string); ok {
									EnvVar = []string{env}
								} else if valueOfSlice, ok := value.([]interface{}); ok {
									for _, elem := range valueOfSlice {
										if env, ok := elem.(string); ok {
											EnvVar = append(EnvVar, env)
										}
									}
								}

							case "extrarunid", "extrarunuid", "extrarun_id", "extrarun_uid", "extra_run_id", "extra_run_uid":
								ExtraRunUID = fmt.Sprintf("%v", value)
							}
						}
						metrics.FetchNewJobLatency.WithLabelValues(
							"job_response_enriched", config.C.PrometheusNamespace, config.C.PrometheusService,
						).Observe(float64(time.Since(start).Nanoseconds()))

						if len(JobId) < 1 {
							continue
						}

						job := model.NewJob(fmt.Sprintf("%v", JobId), CMD)
						job.RunUID = RunUID
						job.ExtraRunUID = ExtraRunUID
						job.RawParams = append(job.RawParams, jobResponse)

						if TTR < 1 {
							TTR = uint64((time.Duration(8*3600) * time.Second).Milliseconds())
						}
						job.TTR = TTR
						job.CmdENV = EnvVar
						metrics.FetchNewJobLatency.WithLabelValues(
							"new_job_created", config.C.PrometheusNamespace, config.C.PrometheusService,
						).Observe(float64(time.Since(start).Nanoseconds()))
						job.StartAt = time.Now()
						if JobsRegistry.Add(job) {
							metrics.FetchNewJobLatency.WithLabelValues(
								"new_job_appended_to_registry", config.C.PrometheusNamespace, config.C.PrometheusService,
							).Observe(float64(time.Since(start).Nanoseconds()))
							metrics.JobsFetchProcessed.Inc()
							jobs <- job
							j += 1
							log.Tracef("sent job id %v ", job.Id)
						} else {
							log.Trace(fmt.Sprintf("Duplicated job id %v ", job.Id))
							metrics.JobsFetchDuplicates.Inc()
						}
					}
				} else {
					log.Tracef("Failed fetch a new Jobs portion due %v ", err)
				}
				metrics.FetchNewJobLatency.WithLabelValues(
					"total", config.C.PrometheusNamespace, config.C.PrometheusService,
				).Observe(float64(time.Since(start).Nanoseconds()))
			}
		}
	}()

	// Single goroutine for canceling jobs
	// We are getting such jobs from API
	// exists on kill

	log.Infof("Fetching jobs for cancellation with delay %v", interval)
	go func() {
		j := 0
		for {
			select {

			case <-ctx.Done():
				doneNumCancelJobs <- j
				log.Debug("Jobs cancellation loop finished [ SUCCESSFULLY ]")

				return
			case <-tickerCancelJobs.C:
				start := time.Now()
				metrics.FetchCancelLatency.WithLabelValues(
					"registry_cleanup", config.C.PrometheusNamespace, config.C.PrometheusService,
				).Observe(float64(time.Since(start).Nanoseconds()))

				stage := "jobs.cancelation"
				params := model.GetAPIParamsFromSection(stage)

				if err, jobsCancellationData := model.DoApiCall(context.WithValue(ctx, model.CtxKeyRequestTimeout, maxRequestTimeout), params, stage); err != nil {
					metrics.FetchCancelLatency.WithLabelValues(
						"failed_query", config.C.PrometheusNamespace, config.C.PrometheusService,
					).Observe(float64(time.Since(start).Nanoseconds()))
					log.Tracef("failed to update api, got: %s and %s", jobsCancellationData, err)
				} else {

					for _, jobResponse := range jobsCancellationData {
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
						jobCancellationId := model.StoreKey(JobId, RunUID, ExtraRunUID)
						if jobForCancellation, ok := JobsRegistry.Record(jobCancellationId); ok {
							metrics.JobsCancelled.Inc()
							if err := jobForCancellation.Cancel(); err != nil {
								log.Tracef("Can't cancel '%s' got %s ", jobCancellationId, err)
							}
						} /* else {
							log.Tracef("Can't find job key '%s' for cancelation: %s extra %s runid %s", jobCancellationId, JobId, ExtraRunUID, RunUID)
							JobsRegistry.Map(func(key string, job *model.Job) {
								log.Tracef("Left Job %s => %p in %s cmd: %s", job.StoreKey(), job, job.Status, job.CMD)
							})
						} */

					}
				}
				metrics.FetchCancelLatency.WithLabelValues(
					"total", config.C.PrometheusNamespace, config.C.PrometheusService,
				).Observe(float64(time.Since(start).Nanoseconds()))

			}

		}

	}()

	numSentJobs := <-doneNumJobs
	numCancelJobs := <-doneNumCancelJobs

	log.Infof("Sent %v jobs", numSentJobs)
	if numCancelJobs > 0 {
		log.Infof("Canceled %v jobs", numCancelJobs)
	}
	return nil
}

// GracefulShutdown cancel all running jobs
// returns error in case any job failed to cancel
func GracefulShutdown(jobs <-chan *model.Job) bool {
	// empty jobs channel
	if len(jobs) > 0 {
		log.Tracef("jobs chan still has size %v, empty it", len(jobs))
		for len(jobs) > 0 {
			<-jobs
		}
	}
	JobsRegistry.GracefullyShutdown()
	if JobsRegistry.Len() > 0 {
		log.Tracef("GracefullyShutdown failed, '%v' jobs left ", JobsRegistry.Len())
		return false
	}
	return true

}
