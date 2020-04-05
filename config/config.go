// Copyright 2020 Valeriy Soloviov. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// license that can be found in the LICENSE file.

// Package config provides configuration for `supraworker` application.

package config

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"runtime"
	"strings"
)

const (
	// Project name
	ProjectName = "supraworker"
)

// top level Configuration structure
type Config struct {
	// Indentification for the process
	ClientId string `mapstructure:"clientId"`
	// delay between API calls to prevent Denial-of-service
	CallAPIDelaySec int `mapstructure:"api_delay_sec"`
	// represent API for Jobs
	JobsAPI ApiOperations `mapstructure:"jobs"`
	// LogsAPI         ApiOperations `mapstructure:"logs"`
	HeartBeat ApiOperations `mapstructure:"heartbeat"`
	// Config version
	ConfigVersion string `mapstructure:"version"`
}

type ApiOperations struct {
	Run         UrlConf `mapstructure:"run"`         // defines how to run item
	Cancelation UrlConf `mapstructure:"cancelation"` // defines how to cancel item

	LogStreams UrlConf `mapstructure:"logstream"` // defines how to get item

	Get    UrlConf `mapstructure:"get"`    // defines how to get item
	Lock   UrlConf `mapstructure:"lock"`   // defines how to lock item
	Update UrlConf `mapstructure:"update"` // defines how to update item
	Unlock UrlConf `mapstructure:"unlock"` // defines how to unlock item
	Finish UrlConf `mapstructure:"finish"` // defines how to finish item
	Failed UrlConf `mapstructure:"failed"` // defines how to update on failed
	Cancel UrlConf `mapstructure:"cancel"` // defines how to update on cancel

}

type UrlConf struct {
	Url             string            `mapstructure:"url"`
	Method          string            `mapstructure:"method"`
	Headers         map[string]string `mapstructure:"headers"`
	PreservedFields map[string]string `mapstructure:"preservedfields"`
	Params          map[string]string `mapstructure:"params"`
}

var (
	CfgFile  string
	ClientId string
	C        Config = Config{
		CallAPIDelaySec: int(2),

		// JobsAPI: UrlConf{
		//     Method: "GET",
		//     Headers: []RequestHeader{
		//         RequestHeader{
		//             Key: "Content-type",
		//             Value: "application/json",
		//         },
		//     },
		// },
	}
	log = logrus.WithFields(logrus.Fields{"package": "config"})
)

// Init configuration
func init() {
	cobra.OnInitialize(initConfig)

}

// ReinitializeConfig on load or file change
func ReinitializeConfig() {
	if len(ClientId) > 0 {
		C.ClientId = ClientId
	}
	if len(C.ClientId) < 1 {
		C.ClientId = "supraworker"
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Don't forget to read config either from CfgFile or from home directory!
	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		lProjectName := strings.ToLower(ProjectName)
		log.Debug("Searching for config with project", ProjectName)
		viper.AddConfigPath(".")
		viper.AddConfigPath("..")
		switch runtime.GOOS {
		case "windows":
			if userprofile := os.Getenv("USERPROFILE"); userprofile != "" {
				viper.AddConfigPath(userprofile)
			}
		default:
			// freebsd, openbsd, darwin, linux
			// plan9, windows...
			viper.AddConfigPath("$HOME/")
			viper.AddConfigPath(fmt.Sprintf("$HOME/.%s/", lProjectName))
			viper.AddConfigPath("/etc/")
			viper.AddConfigPath(fmt.Sprintf("/etc/%s/", lProjectName))

		}

		if conf := os.Getenv(fmt.Sprintf("%s_CFG", strings.ToUpper(ProjectName))); conf != "" {
			viper.SetConfigName(conf)
		} else {
			viper.SetConfigType("yaml")
			viper.SetConfigName(lProjectName)
		}
	}
	viper.AutomaticEnv()
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatal("Can't read config:", err)
	}
	err := viper.Unmarshal(&C)
	if err != nil {
		logrus.Fatal(fmt.Sprintf("unable to decode into struct, %v", err))

	}
	log.Debug(viper.ConfigFileUsed())

}
