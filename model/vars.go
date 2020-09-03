package model

import (
	"github.com/sirupsen/logrus"
	"os/exec"
)

var (
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
}

const (
	JOB_STATUS_PENDING     = "pending"
	JOB_STATUS_IN_PROGRESS = "in_progress"
	JOB_STATUS_SUCCESS     = "success"
	JOB_STATUS_ERROR       = "error"
	JOB_STATUS_CANCELED    = "canceled"
)

func init() {
	previousLevel = logrus.GetLevel()
}
