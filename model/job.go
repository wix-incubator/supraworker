package model

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// IsTerminalStatus returns true if status is terminal:
// - Failed
// - Canceled
// - Successful
func IsTerminalStatus(status string) bool {
	switch status {
	case JOB_STATUS_ERROR, JOB_STATUS_CANCELED, JOB_STATUS_SUCCESS:
		return true
	}
	return false
}

// StoreKey returns Job unique store key
func StoreKey(Id string, RunUID string, ExtraRunUID string) string {
	return fmt.Sprintf("%s:%s:%s", Id, RunUID, ExtraRunUID)
}

// Job public structure
type Job struct {
	Id                    string        // Identificator for Job
	RunUID                string        // Running indentification
	ExtraRunUID           string        // Extra indentification
	Priority              int64         // Priority for a Job
	CreateAt              time.Time     // When Job was created
	StartAt               time.Time     // When command started
	LastActivityAt        time.Time     // When job metadata last changed
	Status                string        // Currentl status
	MaxAttempts           int           // Absoulute max num of attempts.
	MaxFails              int           // Absolute max number of failures.
	TTR                   uint64        // Time-to-run in Millisecond
	CMD                   string        // Comamand
	CmdENV                []string      // Comamand
	RunAs                 string        // RunAs defines user
	ResetBackPresureTimer time.Duration // how often we will dump the logs
	StreamInterval        time.Duration
	mu                    sync.RWMutex
	exitError             error
	ExitCode              int // Exit code
	cmd                   *exec.Cmd
	ctx                   context.Context

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
	// If we should use shell and wrap the command
	UseSHELL   bool
	streamsBuf []string
}

// StoreKey returns StoreKey
func (j *Job) StoreKey() string {
	return StoreKey(j.Id, j.RunUID, j.ExtraRunUID)
}

// GetStatus get job status.
func (j *Job) GetStatus() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status
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

// PutRawParams for all next calls
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
		log.Tracef("Call Canceled for Job %s", j.Id)
		if j.cmd != nil && j.cmd.Process != nil {
			if err := j.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill process: %s", err)
			}
		}

		if errUpdate := j.updateStatus(JOB_STATUS_CANCELED); errUpdate != nil {
			log.Tracef("failed to change job %s status '%s' -> '%s'", j.Id, j.Status, JOB_STATUS_CANCELED)

		}
		j.updatelastActivity()
		stage := "jobs.cancel"
		params := j.GetAPIParams(stage)
		if err, result := DoApiCall(j.ctx, params, stage); err != nil {
			log.Tracef("failed to update api, got: %s and %s", result, err)
		}

	}
	// else {
	// 	log.Trace(fmt.Sprintf("Job %s in terminal '%s' status ", j.Id, j.Status))
	// }
	return nil
}

// Failed job flow
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

		if errUpdate := j.updateStatus(JOB_STATUS_ERROR); errUpdate != nil {
			log.Tracef("failed to change job %s status '%s' -> '%s'", j.Id, j.Status, JOB_STATUS_ERROR)

		}
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

//  for job
// update your API
func (j *Job) AppendLogStream(logStream []string) error {

	if j.quotaHit() {
		<-j.notify
		_ = j.doSendSteamBuf()
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
func (j *Job) resetCounterLoop(ctx context.Context, after time.Duration) {
	ticker := time.NewTicker(after)
	tickerTimeInterval := time.NewTicker(2 * after)
	tickerSlowLogsInterval := time.NewTicker(10 * after)
	defer func() {
		ticker.Stop()
		tickerTimeInterval.Stop()
		tickerSlowLogsInterval.Stop()
		// close(j.notifyLogSent)
	}()
	for {
		select {
		case <-ctx.Done():
			_ = j.doSendSteamBuf()
			// j.notifyLogSent <- struct{}{}
			// log.Tracef("resetCounterLoop finished for '%v'", j.Id)
			return
		case <-j.notifyStopStreams:
			_ = j.doSendSteamBuf()
			// j.notifyLogSent <- struct{}{}
			// log.Tracef("resetCounterLoop finished for '%v'", j.Id)
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
			_ = j.doSendSteamBuf()
		}
	}
}

func (j *Job) doNotify() {
	select {
	case j.notify <- struct{}{}:
	default:
	}
}

// FlushSteamsBuffer - empty current job's streams lines
func (j *Job) FlushSteamsBuffer() error {
	return j.doSendSteamBuf()
}

// doSendSteamBuf low-level functions which sends streams to the remote API
// Send stream only if there is something
func (j *Job) doSendSteamBuf() error {
	j.stremMu.Lock()
	defer j.stremMu.Unlock()
	if len(j.streamsBuf) > 0 {
		// log.Tracef("doSendSteamBuf for '%v' len '%v' %v\n ", j.Id, len(j.streamsBuf),j.streamsBuf)

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
			if _, errReadFrom := buf.ReadFrom(streamsReader); errReadFrom != nil {
				log.Tracef("buf.ReadFrom error %v\n", errReadFrom)
			}
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
	j.mu.Lock()
	ctx, cancel := prepareContaxt(j.ctx, j.TTR)
	defer cancel()
	// Use shell wrapper
	shell, args := CmdWrapper(j.RunAs, j.UseSHELL, j.CMD)
	j.cmd = execCommandContext(ctx, shell, args...)
	j.cmd.Env = MergeEnvVars(j.CmdENV)
	j.mu.Unlock()

	stdout, err := j.cmd.StdoutPipe()
	if err != nil {
		_ = j.AppendLogStream([]string{fmt.Sprintf("cmd.StdoutPipe %s\n", err)})
		return fmt.Errorf("cmd.StdoutPipe, %s", err)
	}

	stderr, err := j.cmd.StderrPipe()
	if err != nil {
		_ = j.AppendLogStream([]string{fmt.Sprintf("cmd.StderrPipe %s\n", err)})
		return fmt.Errorf("cmd.StderrPipe, %s", err)
	}

	err = j.cmd.Start()
	j.mu.Lock()
	if errUpdate := j.updateStatus(JOB_STATUS_IN_PROGRESS); errUpdate != nil {
		log.Tracef("failed to change job %s status '%s' -> '%s'", j.Id, j.Status, JOB_STATUS_IN_PROGRESS)
	}
	j.mu.Unlock()
	if err != nil && j.cmd.Process != nil {
		log.Tracef("Run cmd: %v [%v]\n", j.cmd, j.cmd.Process.Pid)

	} else {
		log.Tracef("Run cmd: %v\n", j.cmd)

	}
	// update API
	stage := "jobs.run"
	if errApi, result := DoApiCall(j.ctx, j.GetAPIParams(stage), stage); errApi != nil {
		log.Tracef("failed to update api, got: %s and %s\n", result, errApi)
	}
	if err != nil {
		_ = j.AppendLogStream([]string{fmt.Sprintf("cmd.Start %s\n", err)})
		return fmt.Errorf("cmd.Start, %s", err)
	}
	notifyStdoutSent := make(chan bool, 1)
	notifyStderrSent := make(chan bool, 1)

	// reset backpresure counter
	per := 5 * time.Second
	if j.ResetBackPresureTimer.Nanoseconds() > 0 {
		per = j.ResetBackPresureTimer
	}
	resetCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go j.resetCounterLoop(resetCtx, per)

	// parse stdout
	// send logs to streaming API
	go func() {
		defer func() {
			notifyStdoutSent <- true
		}()
		stdOutBuf := bufio.NewReader(stdout)
		scanner := bufio.NewScanner(stdOutBuf)
		scanner.Split(bufio.ScanLines)

		buf := make([]byte, 0, 64*1024)
		// The second argument to scanner.Buffer() sets the maximum token size.
		// We will be able to scan the stdout as long as none of the lines is
		// larger than 1MB.
		scanner.Buffer(buf, bufio.MaxScanTokenSize)
		if stdout == nil {
			return
		}
		for scanner.Scan() {
			if errScan := scanner.Err(); errScan != nil {
				stdOutBuf.Reset(stdout)
			}

			msg := scanner.Text()
			_ = j.AppendLogStream([]string{msg, "\n"})
		}

		if scanner.Err() != nil {
			// stdout.Close()
			// log.Tracef("Stdout %v unexpected failure: %v", j.Id, scanner.Err())
			b, err := ioutil.ReadAll(stdout)
			if err == nil {
				_ = j.AppendLogStream([]string{string(b), "\n"})
			} else {
				log.Tracef("Stdout ReadAll %v unexpected failure: %v", j.Id, err)

			}
		}
	}()
	// parse stderr
	// send logs to streaming API
	go func() {
		defer func() {
			notifyStderrSent <- true
		}()

		stdErrScanner := bufio.NewScanner(stderr)
		// stdErrScanner.Split(bufio.ScanWords)
		stdErrScanner.Split(bufio.ScanLines)
		buf := make([]byte, 0, 64*1024)
		// The second argument to scanner.Buffer() sets the maximum token size.
		// We will be able to scan the stdout as long as none of the lines is
		// larger than 1MB.

		stdErrScanner.Buffer(buf, bufio.MaxScanTokenSize)
		if stderr == nil {
			return
		}

		stdErrScanner.Split(bufio.ScanLines)

		for stdErrScanner.Scan() {
			msg := stdErrScanner.Text()
			_ = j.AppendLogStream([]string{fmt.Sprintf("%s\n", msg)})
		}
		if stdErrScanner.Err() != nil {
			log.Tracef("Stderr %v unexpected failure: %v", j.Id, stdErrScanner.Err())
			b, err := ioutil.ReadAll(stderr)
			if err == nil {
				_ = j.AppendLogStream([]string{string(b), "\n"})
			} else {
				log.Tracef("Stderr ReadAll %v unexpected failure: %v", j.Id, err)

			}

		}

	}()
	<-notifyStdoutSent
	<-notifyStderrSent

	// The returned error is nil if the command runs, has
	// no problems copying stdin, stdout, and stderr,
	// and exits with a zero exit status.
	// log.Tracef("cmd.Wait %v", j.Id)
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
		_ = j.AppendLogStream([]string{fmt.Sprintf("%s\n", err)})
		j.exitError = err
	}
	if err == nil {
		signaled := ws.Signaled()
		signal := ws.Signal()
		// log.Tracef("Error: %v", err)
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
		finalStatus := JOB_STATUS_ERROR
		if err == nil {
			finalStatus = JOB_STATUS_SUCCESS
		}

		if errUpdate := j.updateStatus(finalStatus); errUpdate != nil {
			log.Tracef("failed to change job %s status '%s' -> '%s'", j.Id, j.Status, finalStatus)

		}
		log.Tracef("[RUN] Job '%s' is moved to state %s", j.Id, j.Status)
	} else {
		log.Tracef("[RUN] Job '%s' is in terminal state to state %s", j.Id, j.Status)
	}
	// <-j.notifyLogSent
	return err
}

// Finish is triggered when execution is successful.
func (j *Job) Finish() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.updatelastActivity()
	if errUpdate := j.updateStatus(JOB_STATUS_SUCCESS); errUpdate != nil {
		log.Tracef("failed to change job %s status '%s' -> '%s'", j.Id, j.Status, JOB_STATUS_SUCCESS)
	}
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
		RunAs:             "",
		StreamInterval:    time.Duration(5) * time.Second,
	}
}

// NewTestJob return Job with defaults for test
func NewTestJob(id string, cmd string) *Job {
	j := NewJob(id, cmd)
	j.CmdENV = []string{"GO_WANT_HELPER_PROCESS=1"}
	return j
}
