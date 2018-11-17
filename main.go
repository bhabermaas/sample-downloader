// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wercker/ocidownload/downloadserver"
)

// This module is a test module for the download feature. It simply checks for signals and starts the
// POST download server
func main() {

	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			//handle SIGINT
			log.Fatal("SIGINT interrupt")
		case syscall.SIGTERM:
			//handle SIGTERM
			log.Fatal("SIGTERM interrupt")
		}
	}()
	go downloadserver.OCIdownloadServer(8080)
	time.Sleep(time.Hour * 24)
}
