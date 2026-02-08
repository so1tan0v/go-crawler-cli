package main

import (
	cliapp "code/src/infra/cli-app/cli-app"
	"context"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	appName = "hexlet-go-crawler"
	appInfo = "analyze a website structure"
	version = "0.0.1"
)

/*
NAME:
   hexlet-go-crawler - analyze a website structure

USAGE:
   hexlet-go-crawler [global options] command [command options] <url>

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --depth value       crawl depth (default: 10)
   --retries value     number of retries for failed requests (default: 1)
   --delay value       delay between requests (example: 200ms, 1s) (default: 0s)
   --timeout value     per-request timeout (default: 15s)
   --rps value         limit requests per second (overrides delay) (default: 0)
   --user-agent value  custom user agent
   --workers value     number of concurrent workers (default: 4)
   --help, -h          show help
*/

var flags = []cli.Flag{
	&cli.IntFlag{
		Name:    "depth",
		Aliases: []string{"d"},
		Usage:   "crawl depth",
		Value:   10,
	},
	&cli.IntFlag{
		Name:    "retries",
		Aliases: []string{"r"},
		Usage:   "number of retries for failed requests",
		Value:   1,
	},
	&cli.DurationFlag{
		Name:    "delay",
		Aliases: []string{"d"},
		Usage:   "delay between requests",
		Value:   0,
	},
	&cli.DurationFlag{
		Name:    "timeout",
		Aliases: []string{"t"},
		Usage:   "per-request timeout",
		Value:   15 * time.Second,
	},
	&cli.IntFlag{
		Name:    "rps",
		Aliases: []string{"r"},
		Usage:   "limit requests per second",
		Value:   0,
	},
	&cli.StringFlag{
		Name:    "user-agent",
		Aliases: []string{"a"},
		Usage:   "custom user agent",
	},
	&cli.IntFlag{
		Name:    "workers",
		Aliases: []string{"w"},
		Usage:   "number of concurrent workers",
		Value:   4,
	},
}

func main() {
	cliApp := cliapp.NewCliApp()

	if err := cliApp.Init(appName, appInfo, version, flags); err != nil {
		log.Fatalf("Failed to initialize CLI app: %v", err)
	}

	ctx := context.Background()
	args := os.Args

	if err := cliApp.Run(ctx, args); err != nil {
		log.Fatalf("Failed to run CLI app: %v", err)
	}
}
