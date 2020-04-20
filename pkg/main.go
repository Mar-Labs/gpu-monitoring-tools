/*
 * Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"os"
	"syscall"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/urfave/cli/v2"
	"github.com/NVIDIA/gpu-monitoring-tools/bindings/go/dcgm"
)

var (
	BuildVersion = "Filled by the build system"

	CLIFieldsFile = "fields-file"
	CLIPort = "port"

	connectAddr = "localhost"
	isSocket    = "0"
)

func main() {
	c := cli.NewApp()
	c.Name = "DCGM Exporter"
	c.Usage = "Generates GPU metrics in the prometheus format"
	c.Version = BuildVersion

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    CLIFieldsFile,
			Usage:   "Path to the file, that contains the DCGM fields to export",
			Value:   "/etc/dcgm-exporter/default.csv",
			EnvVars: []string{"DCGM_EXPORTER_FIELDS_FILE"},
		},
		&cli.IntFlag{
			Name:    CLIPort,
			Aliases: []string{"p"},
			Value:   8080,
			Usage:   "Port",
			EnvVars: []string{"DCGM_EXPORTER_PORT"},
		},
	}

	c.Action = func(c *cli.Context) error {
		return Run(c)
	}

	if err := c.Run(os.Args); err != nil {
		glog.Fatal(err)
	}
}

func Run(c *cli.Context) error {
	restart:
	defer glog.Flush()

	config := contextToConfig(c)

	cleanup, err := dcgm.Init(dcgm.Standalone, connectAddr, isSocket)
	defer cleanup()
	if err != nil {
		glog.Fatal(err)
	}

	ch := make(chan string, 10)
	pipeline, cleanup, err := NewMetricsPipeline(config)
	defer cleanup()
	if err != nil {
		glog.Fatal(err)
	}

	server, cleanup, err := NewMetricsServer(config, ch)
	defer cleanup()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	stop := make(chan interface{})

	wg.Add(1)
	go pipeline.Run(ch, stop, &wg)

	wg.Add(1)
	go server.Run(stop, &wg)

	sigs := newOSWatcher(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	for {
		select {
		case sig := <-sigs:
			close(stop)
			err := WaitWithTimeout(&wg, time.Second * 2)
			if err != nil {
				glog.Fatal(err)
			}

			if sig == syscall.SIGHUP {
				goto restart
			}

			return nil
		}
	}

	return nil
}

func contextToConfig(c *cli.Context) *Config {
	return &Config {
		FieldsFile: c.String(CLIFieldsFile),
		Port: c.Int(CLIPort),
		CollectInterval: 2000,
	}
}
