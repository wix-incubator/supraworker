package model

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// DefaultPath returns PATH variable
func DefaultPath() string {
	defaultPath := "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	switch runtime.GOOS {
	case "windows":
		defaultPath = "PATH=%PATH%;%SystemRoot%\\system32;%SystemRoot%;%SystemRoot%\\System32\\Wbem"
	}
	return defaultPath
}

// MergeEnvVars merges enviroments variables.
func MergeEnvVars(CmdENVs []string) (uniqueMergedENV []string) {

	mergedENV := append(CmdENVs, os.Environ()...)
	mergedENV = append(mergedENV, DefaultPath())
	unique := make(map[string]bool, len(mergedENV))
	for index := range mergedENV {

		if len(strings.Split(mergedENV[index], "=")) < 2 {
			continue
		}
		k := strings.Split(mergedENV[index], "=")[0]
		if _, ok := unique[k]; !ok {
			uniqueMergedENV = append(uniqueMergedENV, mergedENV[index])
			unique[k] = true
		}
	}
	return
}

// CmdWrapper wraps command.
func CmdWrapper(RunAs string, UseSHELL bool, CMD string) (shell string, args []string) {
	cmdSplit := strings.Fields(CMD)
	if len(RunAs) > 1 {
		shell = "sudo"
		args = []string{"-u", RunAs, CMD}
		switch runtime.GOOS {
		case "windows":
			shell = "runas"
			args = []string{fmt.Sprintf("/user:%s", RunAs), CMD}
		default:
			if bash, err := exec.LookPath("su"); err == nil {
				shell = bash
				args = []string{"-", RunAs, "-c", CMD}
			}
		}

	} else if useCmdAsIs(CMD) {
		shell = cmdSplit[0]
		args = cmdSplit[1:]
	} else if UseSHELL {
		shell = "sh"
		args = []string{"-c", CMD}
		switch runtime.GOOS {
		case "windows":
			if ps, err := exec.LookPath("powershell.exe"); err == nil {
				args = []string{"-NoProfile", "-NonInteractive", CMD}
				shell = ps
			} else if bash, err := exec.LookPath("bash.exe"); err == nil {
				shell = bash
			} else {
				shell = "powershell.exe"
				args = []string{"-NoProfile", "-NonInteractive", CMD}
				log.Tracef("Can't fetch powershell nor bash, got %s\n", err)
			}

		default:
			if bash, err := exec.LookPath("bash"); err == nil {
				shell = bash
			}
		}
	} else {
		shell = cmdSplit[0]
		args = cmdSplit[1:]
	}
	return
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// useCmdAsIs returns true if cmd has shell
func useCmdAsIs(CMD string) bool {
	cmdSplit := strings.Fields(CMD)

	if (len(cmdSplit) > 0) && (fileExists(cmdSplit[0])) {
		// in case of bash
		if strings.HasSuffix(cmdSplit[0], "bash") {
			return true
			// in case su or sudo
			// } else if strings.HasSuffix(cmdSplit[0], "su") || strings.HasSuffix(cmdSplit[0], "sudo") {
			// 	return true
		} else if cmdSplit[0] == "/bin/bash" || cmdSplit[0] == "/bin/sh" {
			return true
		}
	} else if (cmdSplit[0] == "bash") || (cmdSplit[0] == "sh") {
		return true
		// } else if (cmdSplit[0] == "sudo") || (cmdSplit[0] == "su") {
		// 	return true
	}
	return false
}

func urlProvided(stage string) bool {
	url := viper.GetString(fmt.Sprintf("%s.url", stage))
	return len(url) >= 1
}

// prepare context for Job.
func prepareContext(jobCtx context.Context, ttr uint64) (context.Context, context.CancelFunc) {
	var ctx context.Context
	var cancel context.CancelFunc
	// in case we have time limitation or context
	if (ttr > 0) && (jobCtx != nil) {
		ctx, cancel = context.WithTimeout(jobCtx, time.Duration(ttr)*time.Millisecond)
	} else if (ttr > 0) && (jobCtx == nil) {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(ttr)*time.Millisecond)
	} else if jobCtx != nil {
		ctx, cancel = context.WithCancel(jobCtx)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	return ctx, cancel

}
