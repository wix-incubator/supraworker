package model

import (
	// "sync"
	// "fmt"
	// "context"
	"github.com/sirupsen/logrus"
)

// startTrace logs
// works like this in tests:
// startTrace()
// defer restoreLevel()
// func startTrace() {
// 	l := logrus.GetLevel()
// 	if l != logrus.TraceLevel {
// 		previousLevel = l
// 	}
// 	logrus.SetLevel(logrus.TraceLevel)
//
// }

// restore default logLevel
func restoreLevel() {
	logrus.SetLevel(previousLevel)
}
