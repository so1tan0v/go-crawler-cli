package application

import (
	"time"

	"github.com/urfave/cli/v3"
)

/*Флаги для анализа сайта*/
var AppFlags = []cli.Flag{
	&cli.IntFlag{
		Name:  "depth",
		Usage: "crawl depth",
		Value: 10,
	},
	&cli.IntFlag{
		Name:  "retries",
		Usage: "number of retries for failed requests",
		Value: 1,
	},
	&cli.DurationFlag{
		Name:  "delay",
		Usage: "delay between requests (example: 200ms, 1s)",
		Value: 0,
	},
	&cli.DurationFlag{
		Name:  "timeout",
		Usage: "per-request timeout",
		Value: 15 * time.Second,
	},
	&cli.IntFlag{
		Name:  "rps",
		Usage: "limit requests per second (overrides delay)",
		Value: 0,
	},
	&cli.StringFlag{
		Name:  "user-agent",
		Usage: "custom user agent",
		Value: "",
	},
	&cli.IntFlag{
		Name:  "workers",
		Usage: "number of concurrent workers",
		Value: 4,
	},
	&cli.BoolFlag{
		Name:  "indent-json",
		Usage: "pretty-print JSON output",
		Value: false,
	},
}
