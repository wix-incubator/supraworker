package model

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// IsTerminalStatus returns true if status is terminal:
// - Failed
// - Canceled
// - Successfull
func IsTerminalStatus(status string) bool {
	switch status {
	case JOB_STATUS_ERROR, JOB_STATUS_CANCELED, JOB_STATUS_SUCCESS:
		log.Tracef("IsTerminalStatus %s true", status)
		return true
	}
	log.Tracef("IsTerminalStatus %s false", status)
	return false
}

// StoreKey returns Job unique store key
func StoreKey(Id string, RunUID string, ExtraRunUID string) string {
	return fmt.Sprintf("%s:%s:%s", Id, RunUID, ExtraRunUID)
}

type Job struct {
	Id             string
	RunUID         string
	ExtraRunUID    string
	Priority       int64
	CreateAt       time.Time
	StartAt        time.Time
	LastActivityAt time.Time
	Status         string
	MaxAttempts    int      // Absoulute max num of attempts.
	MaxFails       int      // Absolute max number of failures.
	TTR            uint64   // Time-to-run in Millisecond
	CMD            string   // Comamand
	CmdENV         []string // Comamand

	StreamInterval time.Duration
	mu             sync.RWMutex
	exitError      error
	ExitCode       int
	cmd            *exec.Cmd
	ctx            context.Context

	// params got from your API
	RawParams []map[string]interface{}
	// steram interface
	elements          uint
	notify            chan interface{}
	notifyStopStreams chan interface{}
	notifyLogSent     chan interface{}
	stremMu           sync.Mutex
	counter           uint
	timeQuote         bool
	UseSHELL          bool
	streamsBuf        []string
}

// StoreKey returns StoreKey
func (j *Job) StoreKey() string {
	return StoreKey(j.Id, j.RunUID, j.ExtraRunUID)
}

// updatelastActivity for the Job
func (j *Job) updatelastActivity() {
	j.LastActivityAt = time.Now()
}

// updateStatus job status
func (j *Job) updateStatus(status string) error {
	log.Trace(fmt.Sprintf("Job %s status %s -> %s", j.Id, j.Status, status))
	j.Status = status
	return nil
}

// GetRawParams from all previous calls
func (j *Job) GetRawParams() []map[string]interface{} {

	return j.RawParams
}

// GetRawParams from all previous calls
func (j *Job) PutRawParams(params []map[string]interface{}) error {
	j.RawParams = params
	return nil
}

// GetAPIParams for stage from all previous calls
func (j *Job) GetAPIParams(stage string) map[string]string {
	c := make(map[string]string)
	params := GetParamsFromSection(stage, "params")
	for k, v := range params {
		c[k] = v
	}
	resendParamsKeys := GetSliceParamsFromSection(stage, "resend-params")
	// log.Tracef(" GetAPIParams(%s) params params %v\nresend-params %v\n", stage,params,resendParamsKeys)

	for _, resandParamKey := range resendParamsKeys {
		for _, rawVal := range j.RawParams {
			if val, ok := rawVal[resandParamKey]; ok {
				c[resandParamKey] = fmt.Sprintf("%s", val)
			}
		}
	}
	// log.Tracef("GetAPIParams(%s ) c:  %v \n",stage,c)

	return c
}

// Cancel job
// update your API
func (j *Job) Cancel() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !IsTerminalStatus(j.Status) {
		log.Trace(fmt.Sprintf("Call Canceled for Job %s", j.Id))
		if j.cmd != nil && j.cmd.Process != nil {
			if err := j.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %s", err)
			}
		}

		j.updateStatus(JOB_STATUS_CANCELED)
		j.updatelastActivity()
		stage := "jobs.cancel"
		params := j.GetAPIParams(stage)
		if err, result := DoApiCall(j.ctx, params, stage); err != nil {
			log.Tracef("failed to update api, got: %s and %s", result, err)
		}

	} else {
		log.Trace(fmt.Sprintf("Job %s in terminal '%s' status ", j.Id, j.Status))
	}
	return nil
}

// Cancel job
// update your API
func (j *Job) Failed() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !IsTerminalStatus(j.Status) {
		log.Trace(fmt.Sprintf("Call Failed for Job %s", j.Id))

		if j.cmd != nil && j.cmd.Process != nil {
			if err := j.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %s", err)
			}
		}

		j.updateStatus(JOB_STATUS_ERROR)
		log.Tracef("[FAILED] Job '%s' moved to state %s", j.Id, j.Status)

		j.updatelastActivity()
	} else {
		log.Tracef("[FAILED] Job '%s' is in terminal state to state %s", j.Id, j.Status)
	}
	stage := "jobs.failed"
	params := j.GetAPIParams(stage)
	if err, result := DoApiCall(j.ctx, params, stage); err != nil {
		log.Tracef("failed to update api, got: %s and %s", result, err)
	}
	return nil
}

// AppendLogStream for job
// update your API
func (j *Job) AppendLogStream(logStream []string) error {
	if j.quotaHit() {
		<-j.notify
		j.doSendSteamBuf()
	}
	j.incrementCounter()
	j.stremMu.Lock()
	j.streamsBuf = append(j.streamsBuf, logStream...)
	j.stremMu.Unlock()
	return nil
}

//count next element
func (j *Job) incrementCounter() {
	j.stremMu.Lock()
	defer j.stremMu.Unlock()
	j.counter++
}

func (j *Job) quotaHit() bool {
	return (j.counter >= j.elements) || (len(j.streamsBuf) > int(j.elements)) || (j.timeQuote)
}

//scheduled elements counter refresher
func (j *Job) resetCounterLoop(after time.Duration) {
	ticker := time.NewTicker(after)
	tickerTimeInterval := time.NewTicker(2 * after)
	tickerSlowLogsInterval := time.NewTicker(10 * after)
	defer func() {
		ticker.Stop()
		tickerTimeInterval.Stop()
		tickerSlowLogsInterval.Stop()
		close(j.notifyLogSent)
	}()
	for {
		select {
		case <-j.notifyStopStreams:
			j.doSendSteamBuf()
			j.notifyLogSent <- struct{}{}

			log.Tracef("resetCounterLoop finished for '%v'", j.Id)
			return
		case <-ticker.C:

			j.stremMu.Lock()
			if j.quotaHit() {
				// log.Tracef("doNotify for '%v'", j.Id)
				j.timeQuote = false
				j.doNotify()

			}
			j.counter = 0
			j.stremMu.Unlock()
		case <-tickerTimeInterval.C:

			j.stremMu.Lock()
			j.timeQuote = true
			j.stremMu.Unlock()
		// flush Buffer for slow logs
		case <-tickerSlowLogsInterval.C:
			j.doSendSteamBuf()
		}
	}
}

func (j *Job) doNotify() {
	select {
	case j.notify <- struct{}{}:
	default:
	}
}

// FlushSteamsBuffer
func (j *Job) FlushSteamsBuffer() error {
	return j.doSendSteamBuf()
}

// doSendSteamBuf low-level functions which sends streams to the remote API
// Send stream only if there is something
func (j *Job) doSendSteamBuf() error {
	if len(j.streamsBuf) > 0 {
		j.stremMu.Lock()
		// log.Tracef("doSendSteamBuf for '%v' len '%v' %v\n ", j.Id, len(j.streamsBuf),j.streamsBuf)
		defer j.stremMu.Unlock()
		streamsReader := strings.NewReader(strings.Join(j.streamsBuf, ""))
		// update API
		stage := "jobs.logstream"
		params := j.GetAPIParams(stage)
		if urlProvided(stage) {
			// log.Tracef("Using DoApiCall for Streaming")
			params["msg"] = strings.Join(j.streamsBuf, "")
			if errApi, result := DoApiCall(j.ctx, params, stage); errApi != nil {
				log.Tracef("failed to update api, got: %s and %s\n", result, errApi)
			}

		} else {
			var buf bytes.Buffer
			buf.ReadFrom(streamsReader)
			fmt.Printf("Job '%s': %s\n", j.Id, buf.String())
		}
		j.streamsBuf = nil

	}
	return nil
}

// runcmd executes command
// returns error
// supports cancellation
func (j *Job) runcmd() error {
	var ctx context.Context
	var cancel context.CancelFunc
	// in case we have time limitation or context
	if (j.TTR > 0) && (j.ctx != nil) {
		ctx, cancel = context.WithTimeout(j.ctx, time.Duration(j.TTR)*time.Millisecond)
	} else if (j.TTR > 0) && (j.ctx == nil) {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(j.TTR)*time.Millisecond)
	} else if j.ctx != nil {
		ctx, cancel = context.WithCancel(j.ctx)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()
	// Use shell wrapper
	j.mu.Lock()
	cmd_splitted := strings.Fields(j.CMD)
	defaultPath := "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	if useCmdAsIs(j.CMD) {
		j.cmd = execCommandContext(ctx, cmd_splitted[0], cmd_splitted[1:]...)

	} else if j.UseSHELL {
		shell := "sh"
		args := []string{"-c", j.CMD}
		switch runtime.GOOS {
		case "windows":
			defaultPath = "%SystemRoot%\\system32;%SystemRoot%;%SystemRoot%\\System32\\Wbem"

			shell = "powershell.exe"
			if ps, err := exec.LookPath("powershell.exe"); err == nil {
				args = []string{"-NoProfile", "-NonInteractive", j.CMD}
				shell = ps

			} else if bash, err := exec.LookPath("bash.exe"); err == nil {
				shell = bash
			} else {
				log.Tracef("Can't fetch powershell nor bash, got %s\n", err)
			}

		default:
			if bash, err := exec.LookPath("bash"); err == nil {
				shell = bash
			}

		}
		j.cmd = execCommandContext(ctx, shell, args...)
	} else {
		j.cmd = execCommandContext(ctx, cmd_splitted[0], cmd_splitted[1:]...)
	}

	mergedENV := append(j.CmdENV, os.Environ()...)
	mergedENV = append(mergedENV, defaultPath)
	unique := make(map[string]bool, len(mergedENV))

	for indx := range mergedENV {
		if len(strings.Split(mergedENV[indx], "=")) != 2 {
			continue
		}
		k := strings.Split(mergedENV[indx], "=")[0]
		if _, ok := unique[k]; !ok {
			j.cmd.Env = append(j.cmd.Env, mergedENV[indx])
			unique[k] = true
		}
		// if strings.HasPrefix(j.cmd.Env[indx],"PATH="){
		//     j.cmd.Env[indx]
		// }
	}
	j.mu.Unlock()

	stdout, err := j.cmd.StdoutPipe()
	if err != nil {
		j.AppendLogStream([]string{fmt.Sprintf("cmd.StdoutPipe %s\n", err)})
		return fmt.Errorf("cmd.StdoutPipe, %s", err)
	}

	stderr, err := j.cmd.StderrPipe()
	if err != nil {
		j.AppendLogStream([]string{fmt.Sprintf("cmd.StderrPipe %s\n", err)})
		return fmt.Errorf("cmd.StderrPipe, %s", err)
	}

	log.Trace(fmt.Sprintf("Run cmd: %v\n", j.cmd))
	err = j.cmd.Start()
	j.mu.Lock()
	j.updateStatus(JOB_STATUS_IN_PROGRESS)
	j.mu.Unlock()
	// update API
	stage := "jobs.run"
	params := j.GetAPIParams(stage)
	if errApi, result := DoApiCall(j.ctx, params, stage); errApi != nil {
		log.Tracef("failed to update api, got: %s and %s\n", result, errApi)
	}
	if err != nil {
		j.AppendLogStream([]string{fmt.Sprintf("cmd.Start %s\n", err)})
		return fmt.Errorf("cmd.Start, %s", err)
	}
	notifyStdoutSent := make(chan bool)
	notifyStderrSent := make(chan bool)

	// reset backpresure counter
	per := 5 * time.Second
	go j.resetCounterLoop(per)

	// parse stdout
	// send logs to streaming API
	go func() {
		defer func() {
			notifyStdoutSent <- true
		}()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			msg := scanner.Text()
			// log.Tracef("stdout: %s\n", msg)
			j.AppendLogStream([]string{fmt.Sprintf("%s\n", msg)})
		}
	}()
	// parse stderr
	// send logs to streaming API
	go func() {
		defer func() {
			notifyStderrSent <- true
		}()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			msg := scanner.Text()
			// log.Tracef("stderr: %s\n", msg)
			j.AppendLogStream([]string{fmt.Sprintf("%s\n", msg)})
		}
	}()

	<-notifyStdoutSent
	<-notifyStderrSent

	// The returned error is nil if the command runs, has
	// no problems copying stdin, stdout, and stderr,
	// and exits with a zero exit status.
	err = j.cmd.Wait()
	if err != nil {
		log.Tracef("cmd.Wait for '%v' returned error: %v", j.Id, err)
	}

	// signal that we've read all logs
	j.notifyStopStreams <- struct{}{}
	status := j.cmd.ProcessState.Sys()
	ws, ok := status.(syscall.WaitStatus)
	if !ok {
		err = fmt.Errorf("process state Sys() was a %T; want a syscall.WaitStatus", status)
		j.exitError = err
	}
	exitCode := ws.ExitStatus()
	j.ExitCode = exitCode
	if exitCode < 0 {
		err = fmt.Errorf("invalid negative exit status %d", exitCode)
		j.exitError = err
	}
	if exitCode != 0 {
		err = fmt.Errorf("exit code '%d'", exitCode)
		j.AppendLogStream([]string{fmt.Sprintf("%s\n", err)})
		j.exitError = err
	}
	if err == nil {
		signaled := ws.Signaled()
		signal := ws.Signal()
		log.Tracef("Error: %v", err)
		if signaled {
			log.Tracef("Signal: %v", signal)
			err = fmt.Errorf("Signal: %v", signal)
			j.exitError = err
		}
	}
	if err == nil && j.Status == JOB_STATUS_CANCELED {
		err = fmt.Errorf("return error for Canceled Job")
	}
	// log.Tracef("The number of goroutines that currently exist.: %v", runtime.NumGoroutine())
	return err
}

// Run job
// return error in case we have exit code greater then 0
func (j *Job) Run() error {
	j.mu.Lock()
	j.StartAt = time.Now()
	j.updatelastActivity()
	j.mu.Unlock()
	err := j.runcmd()
	j.mu.Lock()
	defer j.mu.Unlock()
	j.exitError = err
	j.updatelastActivity()
	if !IsTerminalStatus(j.Status) {
		if err == nil {
			j.updateStatus(JOB_STATUS_SUCCESS)
		} else {
			j.updateStatus(JOB_STATUS_ERROR)
		}
		log.Tracef("[RUN] Job '%s' is moved to state %s", j.Id, j.Status)
	} else {
		log.Tracef("[RUN] Job '%s' is in terminal state to state %s", j.Id, j.Status)
	}
	// <-j.notifyLogSent
	return err
}

// Finish sucessfull job
// update your API
func (j *Job) Finish() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.updatelastActivity()
	j.updateStatus(JOB_STATUS_SUCCESS)
	stage := "jobs.finish"
	params := j.GetAPIParams(stage)
	if err, result := DoApiCall(j.ctx, params, stage); err != nil {
		log.Tracef("failed to update api, got: %s and %s", result, err)
	}

	return nil
}

// SetContext for job
// in case there is time limit for example
func (j *Job) SetContext(ctx context.Context) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ctx = ctx
}

// NewJob return Job with defaults
func NewJob(id string, cmd string) *Job {
	return &Job{
		Id:                id,
		CreateAt:          time.Now(),
		StartAt:           time.Now(),
		LastActivityAt:    time.Now(),
		Status:            JOB_STATUS_PENDING,
		MaxFails:          1,
		MaxAttempts:       1,
		CMD:               cmd,
		CmdENV:            []string{},
		TTR:               0,
		notify:            make(chan interface{}),
		notifyStopStreams: make(chan interface{}),
		notifyLogSent:     make(chan interface{}),
		RawParams:         make([]map[string]interface{}, 0),
		counter:           0,
		elements:          100,
		UseSHELL:          true,
		StreamInterval:    time.Duration(5) * time.Second,
	}
}

// NewTestJob return Job with defaults for test
func NewTestJob(id string, cmd string) *Job {
	j := NewJob(id, cmd)
	j.CmdENV = []string{"GO_WANT_HELPER_PROCESS=1"}
	return j
}
