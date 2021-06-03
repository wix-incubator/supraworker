package worker

import (
	"github.com/sirupsen/logrus"
)

var (
	// Number of active jobs
	NumActiveJobs int64
	// Number of processed jobs
	NumProcessedJobs int64
	logFields     = logrus.Fields{"package": "worker"}
	log           = logrus.WithFields(logFields)
)
