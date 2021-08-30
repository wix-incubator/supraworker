// Copyright 2020 The Supraworker Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// license that can be found in the LICENSE file.

// Package cmd provides CLI interfaces for the `supraworker` application.
package cmd

import (
	"context"
	"fmt"
	// "github.com/epsagon/epsagon-go/epsagon"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	// communicator "github.com/weldpua2008/supraworker/communicator"
	config "github.com/weldpua2008/supraworker/config"
	heartbeat "github.com/weldpua2008/supraworker/heartbeat"
	job "github.com/weldpua2008/supraworker/job"
	metrics "github.com/weldpua2008/supraworker/metrics"
	model "github.com/weldpua2008/supraworker/model"
	worker "github.com/weldpua2008/supraworker/worker"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	verbose   bool
	traceFlag bool
	pprofFlag bool
	promFlag  = true
	log       = logrus.WithFields(logrus.Fields{"package": "cmd"})
	// maxRequestTimeout is timeout for the cancellation and fetch requests.
	// it's high in order to fetch a huge jsons or when the network is slow.
	maxRequestTimeout = 600 * time.Second
)

func init() {

	// Define Persistent Flags and configuration settings, which, if defined here,
	// will be global for application.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	rootCmd.PersistentFlags().BoolVarP(&traceFlag, "trace", "t", false, "trace")
	rootCmd.PersistentFlags().BoolVarP(&pprofFlag, "pprof", "P", false, "pprof")
	rootCmd.PersistentFlags().BoolVarP(&promFlag, "prometheus", "p", false, "prometheus")

	rootCmd.PersistentFlags().StringVar(&config.ClientId, "clientId", "", "ClientId (default is supraworker)")

	rootCmd.PersistentFlags().IntVarP(&config.NumWorkers, "workers", "w", 0, "Number of workers")
	// local flags, which will only run
	// when this action is called directly.

	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
	// Only log the warning severity or above.
	logrus.SetLevel(logrus.InfoLevel)
}

// This represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "supraworker",
	Short: "Supraworker is abstraction layer around jobs",
	Long: `A Fast and Flexible Abstraction around jobs built with
                love by weldpua2008 and friends in Go.
                Complete documentation is available at github.com/weldpua2008/supraworker/cmd`,
	Version: FormattedVersion(),
	Run: func(cmd *cobra.Command, args []string) {
		stopChan := make(chan os.Signal, 1)
		shutdownChan := make(chan bool, 1)
		signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
		// signal.Notify(stopChan, os.Interrupt)
		//    signal.Notify(stopChan, os.Kill)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // cancel when we are getting the kill signal or exit
		var wg sync.WaitGroup
		log.Infof("Starting %s\n", FormattedVersion())
		go func() {
			sig := <-stopChan
			log.Infof("Shutting down - got %v signal", sig)
			cancel()
			shutdownChan <- true
		}()

		if traceFlag {
			logrus.SetLevel(logrus.TraceLevel)
		} else if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}

		log.Trace("Config file:", viper.ConfigFileUsed())
		delay := int64(viper.GetInt("api_delay_sec"))
		if delay < 1 {
			delay = 1
		}

		apiCallDelay := time.Duration(delay) * time.Second

		// load config
		if errCnf := model.ReinitializeConfig(); errCnf != nil {
			log.Tracef("Failed ReinitializeConfig %v\n", errCnf)
		}
		config.ReinitializeConfig()
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			log.Trace("Config file changed:", e.Name)
			if errCnf := model.ReinitializeConfig(); errCnf != nil {
				log.Tracef("Failed ReinitializeConfig %v\n", errCnf)
			}
			config.ReinitializeConfig()
		})

		healthCheckAddr := config.GetStringTemplatedDefault("healthcheck.listen", ":8080")
		healthCheckUri := config.GetStringTemplatedDefault("healthcheck.uri", "/health/is_alive")
		metrics.StartHealthCheck(healthCheckAddr, healthCheckUri)
		if promFlag || config.GetBool("prometheus.enable") || config.GetBool("prometheus.enabled") {
			prometheusAddr := config.GetStringTemplatedDefault("prometheus.listen", ":8080")
			prometheusUri := config.GetStringTemplatedDefault("prometheus.uri", "/metrics")
			metrics.AddPrometheusMetricsHandler(prometheusAddr, prometheusUri)
		}
		if pprofFlag {
			pprofAddr := config.GetStringTemplatedDefault("pprof.listen", ":8080")
			pprofUri := config.GetStringTemplatedDefault("pprof.uri", "/debug/pprof")
			metrics.AddPProf(pprofAddr, pprofUri)

		}
		//srv := metrics.StartHealthCheck(addr, healthCheckURI)
		//defer metrics.WaitForShutdown(ctx, srv)
		metrics.StartAll()
		defer metrics.StopAll(ctx)

		jobs := make(chan *model.Job, config.C.NumWorkers)

		go func() {
			if err := job.StartGenerateJobs(ctx, jobs, apiCallDelay, maxRequestTimeout); err != nil {
				log.Tracef("StartGenerateJobs returned error %v", err)
			}
		}()

		for w := 1; w <= config.C.NumWorkers; w++ {
			wg.Add(1)
			go worker.StartWorker(w, jobs, &wg)
		}
		heartbeatSection := "heartbeat"
		if config.GetBool(fmt.Sprintf("%v.enable", heartbeatSection)) {
			heartbeatApiCallDelay := config.GetTimeDurationDefault(heartbeatSection, "interval", apiCallDelay)
			go func() {
				if err := heartbeat.StartHeartBeat(ctx, heartbeatSection, heartbeatApiCallDelay); err != nil {
					log.Tracef("StartHeartBeat returned error %v", err)
				}
			}()

		}

		wg.Wait()
		time.Sleep(150 * time.Millisecond)

	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
// return error
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
