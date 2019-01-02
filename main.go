// Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

package main

import (
	"os"

	"github.com/wercker/pkg/log"
	cli "gopkg.in/urfave/cli.v1"
)

// This module provides the shell to run the artifact download server as a separate
// executable for the purpose of handing unmanaged runner artifact downloads.
func main() {
	app := cli.NewApp()

	app.Name = "runner-download"
	app.Copyright = "Copyright (c) 2018, 2019 cOracle and/or its affiliates. All rights reserved."
	app.Usage = "Handle processing of artifact downloads for managed/unmanaged runners"

	app.Version = Version()
	app.Compiled = CompiledAt()
	app.Before = log.SetupLogging
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	}
	app.Commands = []cli.Command{
		serverCommand,
	}
	app.Run(os.Args)
}
