package model

import (
	"fmt"
	"github.com/weldpua2008/supraworker/utils"
	"sync"
	"time"
)

// NewRegistry returns a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		all: make(map[string]*Job),
	}
}

// Registry holds all Job Records.
type Registry struct {
	all map[string]*Job
	mu  sync.RWMutex
}

// Add a job.
// Returns false on duplicate or invalid job id.
func (r *Registry) Add(rec *Job) bool {
	if rec == nil || rec.StoreKey() == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.all[rec.StoreKey()]; ok {
		return false
	}
	r.all[rec.StoreKey()] = rec

	return true
}

// Map function
func (r *Registry) Map(f func(string, *Job)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range r.all {
		f(k, v)
	}
}

// Len returns length of registry.
func (r *Registry) Len() int {
	r.mu.RLock()
	c := len(r.all)
	r.mu.RUnlock()
	return c
}

// Delete a job by job ID.
// Return false if record does not exist.
func (r *Registry) Delete(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.all[id]
	if !ok {
		return false
	}
	delete(r.all, id)

	return true
}

// Cleanup by job TTR.
// Return number of cleaned jobs.
// TODO: Consider new timeout status & flow
//  - Add batch
func (r *Registry) Cleanup() (num int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range r.all {
		if v.HitTimeout() {
			if err := v.Timeout(); err != nil {
				utils.LoggerFromContext(*v.GetContext(), log).Debugf("[TIMEOUT] failed %v, Job started at %v, got %v", err, v.StartAt, err)
			} else {
				utils.LoggerFromContext(*v.GetContext(), log).Tracef("[TIMEOUT] successfully, Job started at %v, TTR %v", v.StartAt, time.Duration(v.TTR)*time.Millisecond)
			}

			delete(r.all, k)
			num += 1
		} else if len(v.Id) < 1 {
			log.Tracef("[EMPTY Job] %v", v)
		}

	}
	return num
}

// GracefullyShutdown is used when we stop the Registry.
// cancel all running & pending job
// return false if we can't cancel any job
func (r *Registry) GracefullyShutdown() bool {
	num := r.Cleanup()
	if num > 0 {
		log.Debugf("Successfully cleanup '%d' jobs", num)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	failed := false
	log.Debug("start GracefullyShutdown")
	for k, v := range r.all {
		msg := fmt.Sprintf("Deleting job %s", v.Id)
		if !IsTerminalStatus(v.GetStatus()) {
			if err := v.Cancel(); err != nil {
				msg = fmt.Sprintf("failed cancel job %s %v", v.Id, err)
				failed = true
			} else {
				msg = fmt.Sprintf("successfully canceled job %s", v.Id)
			}
		}
		log.Debugf(msg)
		delete(r.all, k)
	}
	return failed
}

// Record fetch job by Job ID.
// Follows comma ok idiom
func (r *Registry) Record(jid string) (*Job, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if rec, ok := r.all[jid]; ok {
		return rec, true
	}

	return nil, false
}
