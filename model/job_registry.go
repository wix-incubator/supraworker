package model

import (
	"fmt"
	"github.com/weldpua2008/supraworker/utils"
	"runtime"
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.all {
		f(k, v)
	}
}

// Len returns length of registry.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.all)
}

// Delete a job by job ID.
// Return false if record does not exist.
func (r *Registry) Delete(id string) bool {
	log.Infof("Try remove job %s from registry", id)
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

	if r.Len() > 0 {
		r.mu.Lock()
		copyMap := make(map[string]*Job)
		for k, v := range r.all {
			copyMap[k] = v
		}
		r.mu.Unlock()

		log.Infof("Checking Registry %d", len(r.all))
		for k, v := range copyMap {
			switch {
			case v.HitTimeout():
				if err := v.Timeout(); err != nil {
					utils.LoggerFromContext(*v.GetContext(), log).Debugf("[TIMEOUT] failed %v, Job started at %v, got %v", err, v.StartAt, err)
				} else {
					utils.LoggerFromContext(*v.GetContext(), log).Tracef("[TIMEOUT] successfully, Job started at %v, TTR %v", v.StartAt, time.Duration(v.TTR)*time.Millisecond)
				}
			case len(v.Id) < 1:
				log.Tracef("[EMPTY Job] %v", v)
			case v.IsStuck():
				utils.LoggerFromContext(*v.GetContext(), log).Debug("[STUCK JOB] Cleanup")
			default:
				continue
			}
			if r.Delete(k) {
				num += 1
			}
			runtime.Gosched()
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
