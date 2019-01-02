// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wercker/pkg/log"
	"github.com/wercker/runner-download/downloadserver"
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
		Usage:  "port number for this service",
		EnvVar: "PORT",
	},
	cli.StringFlag{
		Name:   "certfile",
		Usage:  "certificate PEM file for HTTPS",
		EnvVar: "CERT_PEM_FILEFILE",
	},
	cli.StringFlag{
		Name:   "keyfile",
		Usage:  "Key PEM file for HTTPS",
		EnvVar: "KEY_PEM_FILE",
	},
}

var serverAction = func(c *cli.Context) error {
	o, err := parseServerOptions(c)
	if err != nil {
		log.WithError(err).Error("Unable to validate arguments")
		return err
	}

	msg := "Interrupted artifact download server and terminated"
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
	ds := downloadserver.NewDownloadServer()
	ds.Debug = o.Debug
	ds.CertPemFile = o.CertFile
	ds.KeyPemFile = o.KeyFile
	log.Info(fmt.Sprintf("Starting artifact download server, listening on port %d", o.Port))
	err = ds.OCIdownloadServer(o.Port)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

type serverOptions struct {
	Port     int
	CertFile string
	KeyFile  string
	Debug    bool
}

func parseServerOptions(c *cli.Context) (*serverOptions, error) {
	debug := c.GlobalBool("debug")
	port := c.Int("port")
	cert := c.String("certfile")
	keyf := c.String("keyfile")
	if !validPortNumber(port) {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}
	if !validateCredentials(cert, keyf) {
		return nil, errors.New("both --certfile and --keyfile must be specified")
	}

	return &serverOptions{
		Port:     port,
		CertFile: cert,
		KeyFile:  keyf,
		Debug:    debug,
	}, nil
}

// validPortNumber returns true if port is between 0 and 65535.
func validPortNumber(port int) bool {
	return port > 0 && port < 65535
}

// validate all HTTPS stuff is present
func validateCredentials(cert string, keyf string) bool {
	if cert == "" && keyf != "" {
		return false
	}
	if cert != "" && keyf == "" {
		return false
	}
	return true
}
