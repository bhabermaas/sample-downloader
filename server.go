// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wercker/ocidownload/downloadserver"
	"github.com/wercker/pkg/log"
	cli "gopkg.in/urfave/cli.v1"
)

var serverCommand = cli.Command{
	Name:   "server",
	Usage:  "start artifact download server",
	Action: serverAction,
	Flags:  append(serverFlags),
}

var serverFlags = []cli.Flag{
	cli.IntFlag{
		Name:   "port",
		Value:  8091,
		EnvVar: "PORT",
	},
}

var serverAction = func(c *cli.Context) error {
	o, err := parseServerOptions(c)
	if err != nil {
		log.WithError(err).Error("Unable to validate arguments")
		return err
	}

	msg := "Server interrupted and terminated"
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			//handle SIGINT
			log.Fatal(msg)
		case syscall.SIGTERM:
			//handle SIGTERM
			log.Fatal(msg)
		}
	}()

	// Note: DownloadServer structure is populated with OCI credentials taken from the
	// environment. If these are coming from somewhere else then tjhey needc to be supplied
	// after the structure is returned.
	downloadserver.NewDownloadServer()
	log.Info("Starting artifact download server")
	downloadserver.OCIdownloadServer(o.Port)
	return nil
}

type serverOptions struct {
	Port int
}

func parseServerOptions(c *cli.Context) (*serverOptions, error) {
	port := c.Int("port")
	if !validPortNumber(port) {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}
	return &serverOptions{
		Port: port,
	}, nil
}

// validPortNumber returns true if port is between 0 and 65535.
func validPortNumber(port int) bool {
	return port > 0 && port < 65535
}
