package config

import "time"

const TimeoutJobsAfter5MinInTerminalState = 5 * time.Minute
const StopReadJobsOutputAfter5Min = 5 * time.Minute
const TimeoutAppendLogStreams = 10 * time.Minute

var (
	// Set of constants for fetching Job's REST API
	CFG_PREFIX_JOBS                 = "jobs"
	CFG_PREFIX_JOBS_TIMEOUT         = "timeout"
	CFG_PREFIX_JOB_TIMEOUT_DURATION = "job_max_duration"
	CFG_PREFIX_JOBS_FETCHER         = "fetch"

	//CFG_PREFIX_COMMUNICATOR defines parameter in the config for Communicators
	CFG_PREFIX_COMMUNICATOR  = "communicator"
	CFG_PREFIX_COMMUNICATORS = "communicators"
	// HTTP Communicator tuning
	// User for allowed response codes definmition.
	CFG_PREFIX_ALLOWED_RESPONSE_CODES = "codes"
	// Defines backoff prefixes
	// More information at
	//   https://github.com/cenkalti/backoff/blob/v4.0.2/exponential.go#L9
	CFG_PREFIX_BACKOFF = "backoff"

	// MaxInterval caps the RetryInterval and not the randomized interval.
	CFG_PREFIX_BACKOFF_MAXINTERVAL = "maxinterval"
	// After MaxElapsedTime the ExponentialBackOff returns Stop.
	// It never stops if MaxElapsedTime == 0.
	CFG_PREFIX_BACKOFF_MAXELAPSEDTIME  = "maxelapsedtime"
	CFG_PREFIX_BACKOFF_INITIALINTERVAL = "initialinterval"

	CFG_COMMUNICATOR_PARAMS_KEY = "params"
	CFG_INTERVAL_PARAMETER      = "interval"
)
