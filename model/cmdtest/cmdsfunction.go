package cmdtest

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	// log           = logrus.WithFields(logrus.Fields{"package": "model"})
	previousLevel logrus.Level
)

func init() {
	previousLevel = logrus.GetLevel()
}

// StartTrace logs.
// works like this in tests:
// startTrace()
// defer restoreLevel()
func StartTrace() {
	l := logrus.GetLevel()
	if l != logrus.TraceLevel {
		previousLevel = l
	}
	logrus.SetLevel(logrus.TraceLevel)

}

// RestoreLevel restore default logLevel.
func RestoreLevel() {
	logrus.SetLevel(previousLevel)
}

type execFunc func(command string, args ...string) *exec.Cmd

// GetFakeExecCommand returns wrapper for the exec.Cmd.
func GetFakeExecCommand(validator func(string, ...string)) execFunc {
	return func(command string, args ...string) *exec.Cmd {
		validator(command, args...)
		return FakeExecCommand(command, args...)
	}
}

// FakeExecCommand returns wrapped exec.Cmd for tests.
func FakeExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// FakeExecCommandContext returns wrapped exec.Cmd for tests.
func FakeExecCommandContext(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// CMDForTest returns warpped cmd.
func CMDForTest(cmd string) string {
	return fmt.Sprintf("%s -test.run=TestHelperProcess -- '%s'", os.Args[0], cmd)
}

// CMDWapBashForTest returns warpped cmd.
func CMDWapBashForTest(cmd string) string {
	return fmt.Sprintf("%s -test.run=TestHelperProcess -- bash -c '%s'", os.Args[0], cmd)
}

// TestHelperProcess used for catch any command and mock the result
// By default we expect `GO_WANT_HELPER_PROCESS` environment variable
// We will print any word after `echo`
// Control exit code:
// - by providing exit 0
// - by default exit code 127
func TestHelperProcess(t *testing.T) {
	// log.Tracef("Call TestHelperProcess")

	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		// fmt.Fprintf(os.Stdout, fmt.Sprintf("os.Args '%v'",os.Args))
		return
	}
	args := os.Args

	// log.Tracef("Got os.Args %v",os.Args)
	// Previous arguments are tests stuff, that looks like :
	// /tmp/go-build…/…/_test/job_test.test -test.run=TestHelperProcess --
	// cmd, args := args[3], args[4:]
	// cmd := args[3]

	re := regexp.MustCompile(`echo (.+?)`)
	reGenerator := regexp.MustCompile(`generate ([0-9]+)`)
	// Handle the case where args[0] is dir:...
	// TODO: Further Validation
	// if !strings.Contains(strings.Join(args, " "), "bash") {
	// 	fmt.Fprintf(os.Stderr, "Expected command to be 'bash'. Got: '%s' %s", strings.Join(args, " "), args)
	// 	os.Exit(2)
	// }

	exitCode := 127
	switch {
	case strings.Contains(strings.Join(args, " "), "exit 0"):
		exitCode = 0
	}
	// some code here to check arguments perhaps?
	switch {
	case strings.Contains(strings.Join(args, " "), "sleep "):
		// fmt.Fprintf(os.Stdout, "Sleep for 10 seconds")
		time.Sleep(10 * time.Second)
	}
	out := ""
	resGenerator := reGenerator.FindStringSubmatch(strings.Join(args, " "))
	res := re.FindStringSubmatch(strings.Join(args, " "))

	if len(resGenerator) > 1 {
		n := resGenerator[1]
		if i, err := strconv.Atoi(n); err == nil {
			out = strings.Repeat("a", i)

		} else {
			out = n
		}
		// fmt.Fprintf(os.Stdout, n)
	} else if len(res) > 1 {
		out = res[1]
	}

	if (len(out) > 0) && (out != string(' ')) {
		fmt.Fprintf(os.Stdout, "'%v'", out)
	}

	os.Exit(exitCode)
}

// GetFunctionName returns name of the function/
func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
