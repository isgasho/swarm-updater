/*
Copyright 2018 codestation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"megpoid.xyz/go/swarm-updater/log"
)

var blacklist []*regexp.Regexp

var (
	// BuildTime indicates the date when the binary was built (set by -ldflags)
	BuildTime string
	// BuildCommit indicates the git commit of the build
	BuildCommit string
	// AppVersion indicates the application version
	AppVersion = "0.1"
	// BuildNumber indicates the compilation number
	BuildNumber = "0"
)

func run(c *cli.Context) error {
	var schedule string

	if c.IsSet("schedule") {
		schedule = c.String("schedule")
	} else {
		schedule = "@every " + strconv.Itoa(c.Int("interval")) + "s"
	}

	return runCron(schedule, c.Bool("label-enable"))
}

func initialize(c *cli.Context) error {
	if c.IsSet("interval") && c.IsSet("schedule") {
		log.Fatal("Only schedule or interval can be defined, not both")
	}

	if c.Bool("label-enable") && (c.IsSet("blacklist") || c.IsSet("blacklist-regex")) {
		log.Fatal("Do not define a blacklist if label-enable is enabled")
	}

	log.Printf("Starting swarm-updater %s.%s", AppVersion, BuildNumber)

	if len(BuildTime) > 0 {
		log.Printf("Build Time: %s", BuildTime)
	}

	if len(BuildCommit) > 0 {
		log.Printf("Build Commit: %s", BuildCommit)
	}

	err := envConfig(c)
	if err != nil {
		return errors.Wrap(err, "failed to sync environment")
	}

	log.EnableDebug(c.Bool("debug"))

	if c.IsSet("blacklist") {
		list := c.StringSlice("blacklist")
		for _, entry := range list {
			regex, err := regexp.Compile(entry)
			if err != nil {
				return errors.Wrap(err, "failed to compile blacklist regex")
			}

			blacklist = append(blacklist, regex)
		}

		log.Debug("Compiled %d blacklist rules", len(list))
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Usage = "automatically update Docker services"
	app.Version = AppVersion + "." + BuildNumber

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host, H",
			Usage: "docker host",
			Value: "unix:///var/run/docker.sock",
		},
		cli.BoolFlag{
			Name:  "tlsverify, t",
			Usage: "use TLS and verify the server certificate",
		},
		cli.StringFlag{
			Name:  "config, c",
			Usage: "location of the docker config files",
		},
		cli.IntFlag{
			Name:   "interval, i",
			Value:  300,
			Usage:  "poll interval (in seconds)",
			EnvVar: "INTERVAL",
		},
		cli.StringFlag{
			Name:   "schedule, s",
			Usage:  "cron schedule",
			EnvVar: "SCHEDULE",
		},
		cli.BoolFlag{
			Name:   "label-enable, l",
			Usage:  fmt.Sprintf("watch services where %s label is set to true", enabledServiceLabel),
			EnvVar: "LABEL_ENABLE",
		},
		cli.StringSliceFlag{
			Name:   "blacklist, b",
			Usage:  "regular expression to match service names to ignore",
			EnvVar: "BLACKLIST",
		},
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "enable debug logging",
			EnvVar: "DEBUG",
		},
	}

	app.Before = initialize
	app.Action = run

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Unrecoverable error: %s", err.Error())
	}
}
