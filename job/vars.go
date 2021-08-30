package job

import (
	"github.com/sirupsen/logrus"
	"github.com/wix/supraworker/model"
)

var (
	log = logrus.WithFields(logrus.Fields{"package": "job"})
	// Registry for the Jobs
	JobsRegistry = model.NewRegistry()
)
