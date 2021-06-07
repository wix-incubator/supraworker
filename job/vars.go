package job

import (
	"github.com/sirupsen/logrus"
	"github.com/weldpua2008/supraworker/model"
)

var (
	log = logrus.WithFields(logrus.Fields{"package": "job"})
	// Registry for the Jobs
	JobsRegistry = model.NewRegistry()
)
