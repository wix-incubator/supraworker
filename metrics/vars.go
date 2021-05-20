package metrics

import (
	"errors"
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	log                  = logrus.WithFields(logrus.Fields{"package": "metrics"})
	ErrServerListenError = errors.New("HTTP server ListenAndServe: ")
	listenServersStore   = make(map[string]*SrvSession)
	mu                   sync.RWMutex
)
