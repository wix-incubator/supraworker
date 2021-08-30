package model

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/mitchellh/go-ps"
	"github.com/sirupsen/logrus"
	"github.com/wix/supraworker/config"
	"github.com/wix/supraworker/utils"
	"io"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// IsTerminalStatus returns true if status is terminal:
// - Failed
// - Canceled
// - Successful
func IsTerminalStatus(status string) bool {
	switch status {
	case JOB_STATUS_ERROR, JOB_STATUS_CANCELED, JOB_STATUS_SUCCESS, JOB_STATUS_TIMEOUT:
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
	Id                     string // Identification for Job
	RunUID                 string // Running identification
	ExtraRunUID            string // Extra identification
	ExtraSendParams        map[string]string
	Priority               int64         // Priority for a Job
	CreateAt               time.Time     // When Job was created
	StartAt                time.Time     // When command started
	LastActivityAt         time.Time     // When job metadata last changed
	StoppedAt              time.Time     // When job stopped
	CMDExecutionStoppedAt  time.Time     // When execution stopped
	PreviousStatus         string        // Previous Status
	Status                 string        // Currently status
	MaxAttempts            int           // Absolute max num of attempts.
	MaxFails               int           // Absolute max number of failures.
	TTR                    uint64        // Time-to-run in Millisecond
	CMD                    string        // Command
	CmdENV                 []string      // Command
	RunAs                  string        // RunAs defines user
	ResetBackPressureTimer time.Duration // how often we will dump the logs
	StreamInterval         time.Duration
	mu                     sync.RWMutex
	exitError              error
	ExitCode               int // Exit code
	cmd                    *exec.Cmd
	ctx                    context.Context
	alreadyStopped         bool
	killOnce               sync.Once
	startOnce              sync.Once
	stopOnce               sync.Once
	inTerminalState        int32
	stopChan               chan struct{}

	// params got from your API
	RawParams []map[string]interface{}
	// stream interface
	elements          uint64
	notify            chan interface{}
	notifyStopStreams chan interface{}
	notifyLogSent     chan interface{}
	streamsMu         sync.Mutex
	counter           uint64
	timeQuote         uint32
	// If we should use shell and wrap the command
	UseSHELL   bool
	streamsBuf []string
}

// PutInTerminal marks job as in terminal status
func (j *Job) PutInTerminal() {
	atomic.StoreInt32(&(j.inTerminalState), int32(1))
	j.stopOnce.Do(func() {
		j.StoppedAt = time.Now()
	})
}

// IsTerminal returns true in case job entered final state
func (j *Job) IsTerminal() bool {
	return atomic.LoadInt32(&(j.inTerminalState)) != 0
}

// StoreKey returns StoreKey
func (j *Job) StoreKey() string {
	return StoreKey(j.Id, j.RunUID, j.ExtraRunUID)
}

// GetStatus get job status.
func (j *Job) GetStatus() string {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.Status
}

// updatelastActivity for the Job
func (j *Job) updatelastActivity() {
	j.LastActivityAt = time.Now()
}

// updateStatus job status
func (j *Job) updateStatus(status string) error {
	if IsTerminalStatus(status) {
		j.PutInTerminal()
	}
	j.GetLogger().Tracef("'%s' -> '%s'", j.Status, status)
	j.PreviousStatus = j.Status
	j.Status = status
	return nil
}

func (j *Job) GetParamsWithResend(stage string) map[string]interface{} {
	prefix := fmt.Sprintf("%s.%s", config.CFG_PREFIX_JOBS, stage)
	resendParamsKeys := GetSliceParamsFromSection(prefix, "resend-params")

	params := j.GetParams()
	for _, resendParamKey := range resendParamsKeys {
		for _, rawVal := range j.RawParams {
			if val, ok := rawVal[resendParamKey]; ok {
				if _, ok = params[resendParamKey]; !ok {
					params[resendParamKey] = fmt.Sprintf("%s", val)
				}
			}
		}
	}
	//log.Infof("Got resend %v for %s", params, prefix)

	return params
}

// GetPreviousStatus returns Previous Status without JOB_STATUS_RUN_OK and JOB_STATUS_RUN_FAILED
func (j *Job) GetPreviousStatus() string {
	switch j.PreviousStatus {
	case JOB_STATUS_RUN_OK, JOB_STATUS_RUN_FAILED:
		return JOB_STATUS_IN_PROGRESS
	}
	return j.PreviousStatus
}

func (j *Job) GetParams() map[string]interface{} {
	previousStatus := j.Status
	if len(j.PreviousStatus) > 0 {
		previousStatus = j.PreviousStatus
	}
	params := map[string]interface{}{
		"Id":                j.Id,
		"JobId":             j.Id,
		"PreviousStatus":    j.GetPreviousStatus(),
		"JobPreviousStatus": previousStatus,
		"Status":            j.Status,
		"RunUID":            j.RunUID,
		"ExtraRunUID":       j.ExtraRunUID,
		//"StoreKey":          j.StoreKey(),
	}
	if j.ExtraSendParams != nil {
		for k, v := range j.ExtraSendParams {
			if _, ok := params[k]; !ok {
				params[k] = v
			}
		}
	}
	return params
}

// GetRawParams from all previous calls
func (j *Job) GetRawParams() []map[string]interface{} {

	return j.RawParams
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

	for _, resendParamKey := range resendParamsKeys {
		for _, rawVal := range j.RawParams {
			if val, ok := rawVal[resendParamKey]; ok {
				c[resendParamKey] = fmt.Sprintf("%s", val)
			}
		}
	}
	return c
}

// Stops the job process and kills all children processes.
func (j *Job) stopProcess() (cancelError error) {
	var processChildren []int
	if j.cmd != nil && j.cmd.Process != nil && !j.alreadyStopped {
		j.killOnce.Do(func() {
			j.alreadyStopped = true
			j.GetLogger().Tracef("Killing main process %v", j.cmd.Process.Pid)
			processTree, errTree := NewProcessTree()
			if errTree == nil {
				processChildren = processTree.Get(j.cmd.Process.Pid)
			} else {
				j.GetLogger().Warnf("Can't fetch process tree, got %v", errTree)
			}

			if err := j.cmd.Process.Kill(); err != nil {
				runtime.Gosched()
				status := j.cmd.ProcessState.Sys()
				ws, ok := status.(syscall.WaitStatus)
				if ok {
					exitStatus := ws.ExitStatus()
					signaled := ws.Signaled()
					signal := ws.Signal()
					//cancelError = fmt.Errorf("failed to kill process: %s", err)
					if !signaled && exitStatus == 0 {
						cancelError = fmt.Errorf("unexpected: err %v, exitStatus was %v + signal %s, while running: %s", err, exitStatus, signal, j.CMD)
					}
				}

			}
			if processList, err := ps.Processes(); err == nil {
				runtime.Gosched()
				for aux := range processList {
					process := processList[aux]
					if ContainsIntInIntSlice(processChildren, process.Pid()) {
						errKill := syscall.Kill(process.Pid(), syscall.SIGTERM)
						j.GetLogger().Tracef("Killing PID: %d --> Name: %s --> ParentPID: %d [%v]", process.Pid(), process.Executable(), process.PPid(), errKill)
					}
				}
			}
		})
	}
	return cancelError
}

// Cancel job
// It triggers an update for the your API if it's configured
func (j *Job) Cancel() error {
	j.mu.Lock()

	defer func() {
		j.updatelastActivity()
		// Fix race condition
		j.mu.Unlock()
	}()

	if !IsTerminalStatus(j.Status) {
		if errUpdate := j.updateStatus(JOB_STATUS_CANCELED); errUpdate != nil {
			j.GetLogger().Warningf("failed to change status '%s' -> '%s'", j.Status, JOB_STATUS_CANCELED)
		}
		j.doApiCall("cancel")

	} else if j.Status != JOB_STATUS_CANCELED {
		j.GetLogger().Tracef("[CANCEL] already in terminal state %s", j.Status)
	}
	return j.stopProcess()
}

// Failed job flow
// update your API
func (j *Job) Failed() error {
	j.mu.Lock()
	defer func() {
		j.updatelastActivity()
		// Fix race condition
		j.mu.Unlock()
	}()
	if !IsTerminalStatus(j.Status) {
		if errUpdate := j.updateStatus(JOB_STATUS_ERROR); errUpdate != nil {
			j.GetLogger().Warningf("failed to change job %s status '%s' -> '%s'", j.Id, j.Status, JOB_STATUS_ERROR)
		}
		j.doApiCall("failed")
	} else if j.Status != JOB_STATUS_ERROR {
		j.GetLogger().Tracef("[FAILED] already in terminal state %s", j.Status)
	}
	return nil
	//return j.stopProcess()
}

func (j *Job) GetTTR() uint64 {
	return atomic.LoadUint64(&j.TTR)
}
func (j *Job) GetTTRDuration() time.Duration {
	return time.Duration(atomic.LoadUint64(&j.TTR)) * time.Millisecond
}

// IsStuck returns true if job in terminal state for more then TimeoutJobsAfter5MinInTerminalState
func (j *Job) IsStuck() bool {
	now := time.Now()
	if j.IsTerminal() && !j.StoppedAt.IsZero() {
		end := j.StoppedAt.Add(config.TimeoutJobsAfter5MinInTerminalState)
		if now.After(end) {
			return true
		}
	}
	if !j.CMDExecutionStoppedAt.IsZero() {
		end := j.CMDExecutionStoppedAt.Add(config.TimeoutJobsAfter5MinInTerminalState)
		if now.After(end) {
			return true
		}
	}
	return false
}

// HitTimeout returns true if job hit timeout
// always false if TTR is 0
func (j *Job) HitTimeout() bool {
	if j.GetTTR() < 1 {
		return false
	}
	now := time.Now()
	end := j.StartAt.Add(j.GetTTRDuration())
	return now.After(end)
}

// Timeout job flow
// update your API
func (j *Job) Timeout() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !IsTerminalStatus(j.Status) {
		if errUpdate := j.updateStatus(JOB_STATUS_TIMEOUT); errUpdate != nil {
			j.GetLogger().Warningf("failed to change status '%s' -> '%s'", j.Status, JOB_STATUS_ERROR)
		}
		j.updatelastActivity()
		j.doApiCall("timeout")
	} else if j.Status != JOB_STATUS_TIMEOUT {
		j.GetLogger().Tracef("[TIMEOUT] is already in terminal state %s", j.Status)
	}
	return j.stopProcess()
}

func (j *Job) TimeoutWithCancel(duration time.Duration) error {
	errorChan := make(chan error)
	go func() {
		err := j.Timeout()
		errorChan <- err
	}()
	var err error
	select {
	case <-time.After(duration):
		err = fmt.Errorf("%w failed to timeout in %v", ErrJobTimeout, duration)
	case err = <-errorChan:
	}
	return err
}

// Appends log stream to the buffer.
// The content of the buffer will be uploaded to API after:
//  - high volume log producers - after j.elements
//	- after buffer is full
//	- after slow log interval
// TODO: try to use channel
func (j *Job) AppendLogStream(logStream []string) (err error) {
	if j.quotaHit() {
		//<-j.notify
		select {
		case <-j.notify:
		case <-time.After(config.TimeoutAppendLogStreams):
			j.GetLogger().Warningf("Timeout AppendLogStream after %v", config.TimeoutAppendLogStreams)
		}
		err = j.doSendSteamBuf()
	}
	j.incrementCounter()
	if len(logStream) > 0 {
		j.streamsMu.Lock()
		defer j.streamsMu.Unlock()
		j.streamsBuf = append(j.streamsBuf, logStream...)
	}
	return err
}

// count next element
func (j *Job) incrementCounter() {
	atomic.AddUint64(&j.counter, 1)
}

// Checks quota for the buffer
// True - need to send
// False - can wait
func (j *Job) quotaHit() bool {
	return (atomic.LoadUint64(&j.counter) >= j.elements) || (len(j.streamsBuf) > int(j.elements)) || (atomic.LoadUint32(&j.timeQuote) == 1)
}

// Flushes buffer state and resets state of counters.
func (j *Job) resetCounterLoop(ctx context.Context, after time.Duration) {
	if after.Milliseconds() < 1 {
		after = 100 * time.Millisecond
	}
	ticker := time.NewTicker(after)
	tickerTimeInterval := time.NewTicker(2 * after)
	tickerSlowLogsInterval := time.NewTicker(10 * after)
	defer func() {
		ticker.Stop()
		tickerTimeInterval.Stop()
		tickerSlowLogsInterval.Stop()
	}()
	for {
		runtime.Gosched()
		select {
		case <-ctx.Done():
			_ = j.doSendSteamBuf()
			return
		case <-j.notifyStopStreams:
			_ = j.doSendSteamBuf()
			return
		case <-ticker.C:
			j.streamsMu.Lock()
			if j.quotaHit() {
				atomic.StoreUint32(&j.timeQuote, 0)
				j.doNotify()
			}
			atomic.StoreUint64(&j.counter, 0)
			j.streamsMu.Unlock()
		case <-tickerTimeInterval.C:
			j.streamsMu.Lock()
			atomic.StoreUint32(&j.timeQuote, 1)
			j.streamsMu.Unlock()
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
		return
	}
}

// FlushSteamsBuffer - empty current job's streams lines
func (j *Job) FlushSteamsBuffer() error {
	return j.doSendSteamBuf()
}

// doSendSteamBuf low-level function streams to the remote API
// Send stream only if there is something
func (j *Job) doSendSteamBuf() error {
	j.streamsMu.Lock()
	defer j.streamsMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), config.StopReadJobsOutputAfter5Min)
	defer cancel()
	if j.streamsBuf != nil && len(j.streamsBuf) > 0 {
		// j.GetLogger().Tracef("doSendSteamBuf for '%v' len '%v' %v\n ", j.Id, len(j.streamsBuf),j.streamsBuf)

		streamsReader := strings.NewReader(strings.Join(j.streamsBuf, ""))
		// update API
		stage := "jobs.logstream"
		params := j.GetAPIParams(stage)
		if urlProvided(stage) {
			// log.Tracef("Using DoApiCall for Streaming")
			params["msg"] = strings.Join(j.streamsBuf, "")
			doneChan := make(chan bool)
			go func() {
				defer func() {
					doneChan <- true
				}()
				if errApi, result := DoApiCall(ctx, params, stage); errApi != nil {
					j.GetLogger().Tracef("failed to update api, got: %s and %s\n", result, errApi)
				}
			}()

			select {
			case <-doneChan:
			case <-time.After(config.StopReadJobsOutputAfter5Min + 1*time.Minute):
				j.GetLogger().Tracef("Timeout before updated api stage %s", stage)
			}

		} else {
			var buf bytes.Buffer
			if _, errReadFrom := buf.ReadFrom(streamsReader); errReadFrom != nil {
				j.GetLogger().Tracef("buf.ReadFrom error %v\n", errReadFrom)
			}
			fmt.Printf("Job '%s': %s\n", j.Id, buf.String())
		}
		j.streamsBuf = nil

	}
	return nil
}

// Run job
// return error in case we have exit code greater then 0
func (j *Job) Run() error {
	j.mu.Lock()
	alreadyRunning := j.Status == JOB_STATUS_IN_PROGRESS || IsTerminalStatus(j.Status)
	ctx, cancel := prepareContext(j.ctx, j.TTR)
	defer cancel()
	j.mu.Unlock()

	if !alreadyRunning {
		j.startOnce.Do(func() {
			j.StartAt = time.Now()
			j.updatelastActivity()
			// Use shell wrapper
			shell, args := CmdWrapper(j.RunAs, j.UseSHELL, j.CMD)
			j.cmd = execCommandContext(ctx, shell, args...)
			j.cmd.Env = MergeEnvVars(j.CmdENV)

		},
		)
	} else {
		return fmt.Errorf("Cannot start Job %s with status '%s' ", j.Id, j.Status)
	}
	err := j.runcmd(ctx)
	j.mu.Lock()
	defer j.mu.Unlock()
	if err != nil && j.exitError == nil {
		j.exitError = err
	}
	j.updatelastActivity()
	return err
}

// runcmd executes command
// returns error
// supports cancellation
func (j *Job) runcmd(ctx context.Context) error {

	// reset backpressure counter
	per := 5 * time.Second
	if j.ResetBackPressureTimer.Nanoseconds() > 0 {
		per = j.ResetBackPressureTimer
	}
	resetCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go j.resetCounterLoop(resetCtx, per)

	stdout, err := j.cmd.StdoutPipe()
	if err != nil {
		msg := fmt.Sprintf("Cannot initial stdout %s\n", err)
		_ = j.AppendLogStream([]string{msg})
		return fmt.Errorf(msg)
	}

	stderr, err := j.cmd.StderrPipe()
	if err != nil {
		msg := fmt.Sprintf("Cannot initial stderr %s\n", err)
		_ = j.AppendLogStream([]string{msg})
		return fmt.Errorf(msg)
	}

	j.mu.Lock()
	err = j.cmd.Start()
	if errUpdate := j.updateStatus(JOB_STATUS_IN_PROGRESS); errUpdate != nil {
		j.GetLogger().Tracef("failed to change status '%s' -> '%s'", j.Status, JOB_STATUS_IN_PROGRESS)
	}

	if err != nil && j.cmd.Process != nil {
		j.GetLogger().Tracef("Start CMD: %s [%d] TTR %v\n", j.cmd, j.cmd.Process.Pid, j.GetTTRDuration())
	} else {
		j.GetLogger().Tracef("Start CMD: %s TTR %v\n", j.cmd, j.GetTTRDuration())
	}
	// update API
	j.doApiCall("run")
	j.mu.Unlock()
	if err != nil {
		j.GetLogger().Tracef("Exiting...cmd.Start has error %s", err)

		_ = j.AppendLogStream([]string{fmt.Sprintf("cmd.Start %s\n", err)})
		return fmt.Errorf("cmd.Start, %s", err)
	}
	notifyStdoutSent := make(chan bool, 1)
	notifyStderrSent := make(chan bool, 1)

	// copies stdout/stderr to to streaming API
	copyStd := func(data *io.ReadCloser, processed chan<- bool) {
		defer func() {
			processed <- true
		}()
		if data == nil {
			return
		}
		stdOutBuf := bufio.NewReader(*data)
		scanner := bufio.NewScanner(stdOutBuf)
		scanner.Split(bufio.ScanLines)

		buf := make([]byte, 0, 64*1024)
		// The second argument to scanner.Buffer() sets the maximum token size.
		// We will be able to scan the stdout as long as none of the lines is
		// larger than 1MB.
		scanner.Buffer(buf, bufio.MaxScanTokenSize)

		for scanner.Scan() {
			if errScan := scanner.Err(); errScan != nil {
				stdOutBuf.Reset(*data)
			}

			msg := scanner.Text()
			_ = j.AppendLogStream([]string{msg, "\n"})
			runtime.Gosched()
		}

		if scanner.Err() != nil {
			b, err := ioutil.ReadAll(*data)
			if err == nil {
				_ = j.AppendLogStream([]string{string(b), "\n"})
			}
			//else {
			//	j.GetLogger().Tracef("[Job %v] Scanner got unexpected failure: %v", j.Id, err)
			//}
		}
	}

	// send stdout to streaming API
	go copyStd(&stdout, notifyStdoutSent)
	runtime.Gosched()

	// send stderr to streaming API
	go copyStd(&stderr, notifyStderrSent)
	runtime.Gosched()
	err = j.cmd.Wait()
	select {
	case <-notifyStdoutSent:
	case <-time.After(config.StopReadJobsOutputAfter5Min):
	}

	select {
	case <-notifyStderrSent:
	case <-time.After(config.StopReadJobsOutputAfter5Min):
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	// The returned error is nil if the command runs, has
	// no problems copying stdin, stdout, and stderr,
	// and exits with a zero exit status.

	j.CMDExecutionStoppedAt = time.Now()
	// signal that we've read all logs
	j.GetLogger().Tracef("Brodcast stop logs")
	j.notifyStopStreams <- struct{}{}
	//j.mu.Unlock()
	//
	//j.mu.Lock()
	//defer j.mu.Unlock()
	status := j.cmd.ProcessState.Sys()
	ws, ok := status.(syscall.WaitStatus)
	exitCode := 127
	if ok {
		exitCode = ws.ExitStatus()
		if err != nil && !ws.Signaled() {
			j.GetLogger().Tracef("cmd.Wait for '%v' returned error: %v", j.Id, err)
		}
	}

	j.ExitCode = exitCode
	j.alreadyStopped = true

	switch {
	case j.Status == JOB_STATUS_CANCELED:
		err = ErrJobCancelled
	case j.Status == JOB_STATUS_TIMEOUT:
		err = ErrJobTimeout
	case ctx != nil && (ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled):
		err = ErrJobTimeout
	case exitCode < 0:
		err = fmt.Errorf("%w %d", ErrInvalidNegativeExitCode, exitCode)
		//_ = j.AppendLogStream([]string{fmt.Sprintf("%s\n", err)})
	case exitCode != 0:
		err = fmt.Errorf("exit code '%d'", exitCode)
		//_ = j.AppendLogStream([]string{fmt.Sprintf("%s\n", err)})
	case err == nil && ok:
		if ws.Signaled() {
			signal := ws.Signal()
			err = fmt.Errorf("%w %v", ErrJobGotSignal, signal)
		}
	case !ok:
		err = fmt.Errorf("%w got %T", ErrorJobNotInWaitStatus, status)
	}

	if !IsTerminalStatus(j.Status) {
		j.Status = JOB_STATUS_RUN_OK
		if err != nil {
			j.Status = JOB_STATUS_RUN_FAILED
		}
	}
	j.exitError = err
	return err
}

// Finish is triggered when execution is successful.
func (j *Job) Finish() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.updatelastActivity()

	if errUpdate := j.updateStatus(JOB_STATUS_SUCCESS); errUpdate != nil {
		j.GetLogger().Tracef("failed to change status '%s' -> '%s'", j.Status, JOB_STATUS_SUCCESS)
	}

	j.doApiCall("finish")
	return nil
}

func (j *Job) doApiCall(stage string) {

	params := j.GetParamsWithResend(stage)
	ctx := j.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := DoApi(ctx, params, stage); err != nil {
		j.GetLogger().Tracef("doApiCall [%s] fails with '%s'", stage, err)
	}
}

// SetContext for job
// in case there is time limit for example
func (j *Job) SetContext(ctx context.Context) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.ctx = ctx
}

// AddToContext for job
// in case there is time limit for example
func (j *Job) AddToContext(key interface{}, value interface{}) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.ctx != nil {
		j.ctx = context.WithValue(j.ctx, key, value)
	}
}

// GetContext of the job
func (j *Job) GetContext() *context.Context {
	//j.mu.RLock()
	//defer j.mu.RUnlock()
	if j.ctx == nil {
		ctx := context.Background()
		return &ctx
	}
	return &j.ctx
}

// GetLogger from job context
func (j *Job) GetLogger() *logrus.Entry {
	return utils.LoggerFromContext(j.ctx, log)
}

// NewJob return Job with defaults
func NewJob(id string, cmd string) *Job {
	return &Job{
		Id:                id,
		CreateAt:          time.Now(),
		StartAt:           time.Now(),
		LastActivityAt:    time.Now(),
		Status:            JOB_STATUS_PENDING,
		PreviousStatus:    JOB_STATUS_PENDING,
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
		ctx:               utils.FromJobID(context.Background(), id),
		StreamInterval:    time.Duration(5) * time.Second,
		stopChan:          make(chan struct{}, 1),
	}
}

// NewTestJob return Job with defaults for test
func NewTestJob(id string, cmd string) *Job {
	j := NewJob(id, cmd)
	j.CmdENV = []string{"GO_WANT_HELPER_PROCESS=1"}
	return j
}
