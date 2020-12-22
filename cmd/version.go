// Copyright 2020 Valeriy Soloviov. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// license that can be found in the LICENSE file.

// Package cmd defines commaqnd tools.
// Version tools for Supraworker
package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.SetVersionTemplate(`{{.Version}}{{printf "\n" }}`)
}

// GitCommit defines the git commit that was used at runtime. This will be filled in by the compiler.
var GitCommit string

const (
	// Version defines main version number
	Version = "0.3.2"

	// VersionPrerelease is pre-release marker for the version
	// such as "dev" (in development), "beta", "rc1", etc.
	VersionPrerelease = "dev"
)

// FormattedVersion returns formatted version as string.
func FormattedVersion() string {
	var versionString bytes.Buffer
	fmt.Fprintf(&versionString, "Supraworker v")
	fmt.Fprintf(&versionString, "%s", Version)
	if VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", VersionPrerelease)

		if GitCommit != "" {
			fmt.Fprintf(&versionString, " (%s)", GitCommit)
		}
	}

	return versionString.String()
}

// version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Supraworker",
	Long:  `All software has versions. This is Supraworker's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(FormattedVersion())
	},
}
