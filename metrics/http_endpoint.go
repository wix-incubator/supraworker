package metrics

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"
)

// SrvSession stores all information about server
type SrvSession struct {
	srv        *http.Server
	mux        *http.ServeMux
	listenAddr string
	isStarted  bool
	mu         sync.RWMutex
}

// IsStarted return whether it was started
func (s *SrvSession) IsStarted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isStarted
}

// Start listen on assigned address
func (s *SrvSession) Start() *http.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Warningf("%v %v", ErrServerListenError, err)
		}
	}()
	return s.srv

}

// Shutdown should be called on exit
func (s *SrvSession) Shutdown(ctx context.Context) {
	if s.srv != nil {
		if err := s.srv.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			log.Warningf("Shutdown for [%v] %v", s.listenAddr, err)
		}
	}
}

func Get(listenAddr string) *SrvSession {
	mu.Lock()
	defer mu.Unlock()
	if srv, ok := listenServersStore[listenAddr]; ok {
		if srv != nil {
			return srv
		}
	}
	listenServersStore[listenAddr] = NewSrvSession(listenAddr)
	return listenServersStore[listenAddr]
}

func StartAll() {
	mu.Lock()
	defer mu.Unlock()
	for _, srv := range listenServersStore {
		if srv != nil {
			srv.Start()
		}
	}
}

func StopAll(ctx context.Context) {
	mu.Lock()
	defer mu.Unlock()
	for _, srv := range listenServersStore {
		if srv != nil {
			srv.Shutdown(ctx)
		}
	}
}

func NewSrvSession(listenAddr string) *SrvSession {
	mux := http.NewServeMux()
	return &SrvSession{
		mux:        mux,
		listenAddr: listenAddr,
		srv: &http.Server{
			Addr:         listenAddr,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      mux,
		},
	}
}
func (s *SrvSession) AddHandler(uri string, h http.Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mux.Handle(uri, h)
}
func (s *SrvSession) AddHandleFunc(uri string, h func(http.ResponseWriter, *http.Request)) {

	s.mu.Lock()
	defer s.mu.Unlock()
	s.mux.HandleFunc(uri, h)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func AddPrometheusMetricsHandler(listenAddr string, uri string) {
	log.Tracef("Start prometheus at %s [%s]", uri, listenAddr)
	srv := Get(listenAddr)
	srv.AddHandler(uri, promhttp.Handler())
}

func AddPProf(listenAddr string, uri string) {
	log.Tracef("Start pprof at %s [%s]", uri, listenAddr)
	srv := Get(listenAddr)
	srv.AddHandleFunc(fmt.Sprintf("%s/", uri), pprof.Index)
	srv.AddHandleFunc(fmt.Sprintf("%s/cmdline", uri), pprof.Cmdline)
	srv.AddHandleFunc(fmt.Sprintf("%s/profile", uri), pprof.Profile)
	srv.AddHandleFunc(fmt.Sprintf("%s/symbol", uri), pprof.Symbol)
	srv.AddHandleFunc(fmt.Sprintf("%s/trace", uri), pprof.Trace)

	srv.srv.ReadTimeout = 120 * time.Second
	srv.srv.WriteTimeout = 120 * time.Second
}

func StartHealthCheck(listenAddr string, uri string) {
	log.Tracef("Start metrics at %s [%s]", uri, listenAddr)
	srv := Get(listenAddr)
	srv.AddHandleFunc(uri, healthHandler)
}

func WaitForShutdown(ctx context.Context, srvSlice []*http.Server) {
	for _, srv := range srvSlice {
		if srv != nil {
			if err := srv.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
				log.Warningf("Shutdown %v", err)
			}
		}
	}
}
