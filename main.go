// Copyright 2020 The Supraworker Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// license that can be found in the LICENSE file.

// This is the main package for the `supraworker` application.

package main

import (
	"github.com/wix/supraworker/cmd"
	"log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
