package domain

import (
	"time"

	"github.com/urfave/cli/v3"
)

var Flags = []cli.Flag{
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
