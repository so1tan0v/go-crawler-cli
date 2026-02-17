package application

import (
	"code/crawler"
	"code/src/domain"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
)

func Action(ctx context.Context, cmd *cli.Command) error {
	targetURL := cmd.Args().First()
	targetURL = strings.TrimPrefix(targetURL, "URL=")
	targetURL = strings.TrimPrefix(targetURL, "url=")

	if targetURL == "" {
		fmt.Fprintln(os.Stderr, "URL is required. Example: hexlet-go-crawler https://example.com")
		_ = cli.ShowAppHelp(cmd)

		return nil
	}

	delay := cmd.Duration("delay")
	rps := cmd.Int("rps")

	opts := domain.Options{
		URL:         targetURL,
		Depth:       cmd.Int("depth"),
		Retries:     cmd.Int("retries"),
		Delay:       delay,
		RPS:         rps,
		Timeout:     cmd.Duration("timeout"),
		UserAgent:   cmd.String("user-agent"),
		Concurrency: cmd.Int("workers"),
		IndentJSON:  cmd.Bool("indent-json"),
		HTTPClient:  &http.Client{},
	}

	report, _ := crawler.Analyze(ctx, opts)
	if len(report) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", string(report))
	}

	return nil
}
