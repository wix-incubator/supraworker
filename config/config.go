// Copyright 2020 Valeriy Soloviov. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// license that can be found in the LICENSE file.

// Package config provides configuration for `supraworker` application.
package config

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"html/template"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	// ProjectName defines project name
	ProjectName = "supraworker"
)

// Config is top level Configuration structure
type Config struct {
	// Indentification for the process
	ClientId      string `mapstructure:"clientId"`
	NumActiveJobs int    // Number of jobs
	NumFreeSlots  int    // Number of free jobs slots
	NumWorkers    int

	ClusterId   string
	ClusterPool string
	URL         string // Used for overload URL for Tests "{{.URL}}"

	// delay between API calls to prevent Denial-of-service
	CallAPIDelaySec int `mapstructure:"api_delay_sec"`
	// represent API for Jobs
	JobsAPI ApiOperations `mapstructure:"jobs"`
	// LogsAPI         ApiOperations `mapstructure:"logs"`
	HeartBeat ApiOperations `mapstructure:"heartbeat"`
	// Config version
	ConfigVersion string `mapstructure:"version"`
}

// ApiOperations is defines operations structure
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

// UrlConf defines all params for request.
type UrlConf struct {
	Url             string            `mapstructure:"url"`
	Method          string            `mapstructure:"method"`
	Headers         map[string]string `mapstructure:"headers"`
	PreservedFields map[string]string `mapstructure:"preservedfields"`
	Params          map[string]string `mapstructure:"params"`
}

var (
	// CfgFile defines Path to the config
	CfgFile string
	// ClientId defines Indentification for the instance.
	ClientId string
	// C defines main configuration structure.
	C Config = Config{
		CallAPIDelaySec: int(2),
		NumActiveJobs:   0,
		NumFreeSlots:    0,
		NumWorkers:      0,
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

func GetStringTemplatedDefault(section string, def string) string {
	v := viper.GetString(section)
	if len(v) > 0 {
		var tplBytes bytes.Buffer

		tpl, err1 := template.New("params").Parse(v)
		if err1 != nil {
			return def
		}
		err := tpl.Execute(&tplBytes, C)
		if err != nil {
			return def
		}
		return tplBytes.String()

	}
	return def
}

// GetMapStringMapStringTemplatedDefault returns map of [string]string maps templated & enriched by default.
func GetMapStringMapStringTemplatedDefault(section string, param string, def map[string]string) map[string]map[string]string {
	ret := make(map[string]map[string]string, 0)
	sections_values := viper.GetStringMap(fmt.Sprintf("%s.%s", section, param))
	for subsection, section_value := range sections_values {
		if section_value == nil {
			continue
		}
		// log.Infof("%s.%s => %v",  section, param,k1)
		if params, ok := section_value.(map[string]interface{}); ok {
			c := make(map[string]string)
			for k, v := range def {
				c[k] = v
			}
			for k, v := range params {
				var tplBytes bytes.Buffer
				tpl, err1 := template.New("params").Parse(fmt.Sprintf("%v", v))
				if err1 != nil {
					continue
				}

				// tpl := template.Must(template.New("params").Parse(fmt.Sprintf("%v", v)))
				if err := tpl.Execute(&tplBytes, C); err != nil {
					// log.Tracef("params executing template for %v got %s", v, err)
					continue
				}
				c[k] = tplBytes.String()
			}
			ret[fmt.Sprintf("%s.%s.%s", section, param, subsection)] = c
		}

	}
	return ret
}
func GetStringMapStringTemplatedDefault(section string, param string, def map[string]string) map[string]string {
	c := make(map[string]string)
	for k, v := range def {
		c[k] = v
	}
	params := viper.GetStringMapString(fmt.Sprintf("%s.%s", section, param))
	for k, v := range params {

		var tplBytes bytes.Buffer
		// WARNING: will panic:
		// tpl := template.Must(template.New("params").Parse(v))
		// we can preserve failed templated string
		c[k] = v
		tpl, err1 := template.New("params").Parse(v)
		if err1 != nil {
			continue
		}
		err := tpl.Execute(&tplBytes, C)
		if err != nil {
			log.Tracef("params executing template: %s", err)
			continue
		}
		c[k] = tplBytes.String()
	}
	return c
}

// GetStringDefault return section string or default.
func GetStringDefault(section string, def string) string {
	if val := viper.GetString(section); len(val) > 0 {
		return val
	}
	return def
}

// GetBool return section string or default.
func GetBool(section string) bool {
	return viper.GetBool(section)
}

func GetStringMapStringTemplated(section string, param string) map[string]string {
	c := make(map[string]string)
	return GetStringMapStringTemplatedDefault(section, param, c)
}

// GetIntSlice returns []int or default.
func GetIntSlice(section string, param string, def []int) []int {
	if val := viper.GetIntSlice(fmt.Sprintf("%v.%v", section, param)); val != nil && len(val) > 0 {
		return val
	}
	return def
}

func GetStringMapStringTemplatedFromMap(section string, param string, from map[string]string) map[string]string {
	c := make(map[string]string)
	return GetStringMapStringTemplatedFromMapDefault(section, param, from, c)
}

func GetStringMapStringTemplatedFromMapDefault(section string, param string, from map[string]string, def map[string]string) map[string]string {
	c := make(map[string]string)
	for k, v := range def {
		c[k] = v
	}
	params := viper.GetStringMapString(fmt.Sprintf("%s.%s", section, param))
	for k, v := range params {

		var tplBytes bytes.Buffer
		// WARNING: will panic:
		// tpl := template.Must(template.New("params").Parse(v))
		// we can preserve failed templated string
		c[k] = v
		tpl, err1 := template.New("params").Parse(v)
		if err1 != nil {
			continue
		}
		err := tpl.Execute(&tplBytes, from)
		if err != nil {
			log.Tracef("Failed params executing template: %s from : %v", err, from)
			continue
		}
		c[k] = tplBytes.String()
	}
	return c
}

// GetTimeDuration return delay for the section with default of 1 second.
// Example config:
// section:
//     interval: 5s
func GetTimeDurationDefault(section string, param string, def time.Duration) (interval time.Duration) {
	var comp time.Duration

	for _, k := range []string{fmt.Sprintf("%v.%v.%v", section, param, CFG_INTERVAL_PARAMETER),
		fmt.Sprintf("%v.%v", section, param), section, CFG_INTERVAL_PARAMETER} {
		delay := viper.GetDuration(k)
		if delay > comp && delay.Milliseconds() > 0 {
			return delay
		}
	}
	return def
}

func ConvertMapStringToInterface(in map[string]string) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range in {
		out[k] = v
	}
	return out
}
