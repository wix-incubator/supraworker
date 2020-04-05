package model

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
)

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func useCmdAsIs(CMD string) bool {
	cmd_splitted := strings.Fields(CMD)

	if (len(cmd_splitted) > 0) && (fileExists(cmd_splitted[0])) {
		// in case of bash
		if strings.HasSuffix(cmd_splitted[0], "bash") {
			return true
			// in case su or sudo
			// } else if strings.HasSuffix(cmd_splitted[0], "su") || strings.HasSuffix(cmd_splitted[0], "sudo") {
			// 	return true
		} else if cmd_splitted[0] == "/bin/bash" || cmd_splitted[0] == "/bin/sh" {
			return true
		}
	} else if (cmd_splitted[0] == "bash") || (cmd_splitted[0] == "sh") {
		return true
		// } else if (cmd_splitted[0] == "sudo") || (cmd_splitted[0] == "su") {
		// 	return true
	}
	return false
}

func urlProvided(stage string) bool {
	url := viper.GetString(fmt.Sprintf("%s.url", stage))
	if len(url) < 1 {
		return false
	}
	return true
}
