package model

import (
	"errors"
	"github.com/sirupsen/logrus"
	"os/exec"
)

type ContextKey string

const (
	CtxKeyRequestTimeout ContextKey = "ctx_req_timeout"
	CtxKeyRequestWorker  ContextKey = "ctx_req_worker_id"
)

var (
	ErrFailedSendRequest       = errors.New("Failed to send request")
	ErrInvalidNegativeExitCode = errors.New("Invalid negative exit code")
	ErrJobCancelled            = errors.New("Job cancelled")
	ErrJobTimeout              = errors.New("Job timeout")
	ErrJobGotSignal            = errors.New("Job got sgnal")
	ErrorJobNotInWaitStatus    = errors.New("Process state is not in WaitStatus")

	execCommandContext = exec.CommandContext
	// FetchNewJobAPIURL is URL for pulling new jobs
	FetchNewJobAPIURL string
	// FetchNewJobAPIMethod is Http METHOD for fetch Jobs API
	FetchNewJobAPIMethod = "POST"
	// FetchNewJobAPIParams is used in eqch requesto for a new job
	FetchNewJobAPIParams = make(map[string]string)

	// StreamingAPIURL is URL for uploading log steams
	StreamingAPIURL string
	// StreamingAPIMethod is Http METHOD for streaming log API
	StreamingAPIMethod = "POST"

	log           = logrus.WithFields(logrus.Fields{"package": "model"})
	previousLevel logrus.Level
)

// Jobber defines a job interface.
type Jobber interface {
	Run() error
	Cancel() error
	Finish() error
	Timeout() error
}

const (
	JOB_STATUS_PENDING     = "PENDING"
	JOB_STATUS_IN_PROGRESS = "RUNNING"
	JOB_STATUS_SUCCESS     = "SUCCESS"
	JOB_STATUS_RUN_OK      = "RUN_OK"
	JOB_STATUS_RUN_FAILED  = "RUN_FAILED"
	JOB_STATUS_ERROR       = "ERROR"
	JOB_STATUS_CANCELED    = "CANCELED"
	JOB_STATUS_TIMEOUT     = "TIMEOUT"
)

func init() {
	previousLevel = logrus.GetLevel()
}
