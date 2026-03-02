package main

import (
	"code/internal/application"
	cliapp "code/internal/infra/cli-app"

	"context"
	"fmt"
	"os"
)

const (
	appName = "so1-crawler"
	appInfo = "analyze a website structure"
	version = "1.1.0"
)

func main() {
	cliApp := cliapp.NewCliApp()

	cliApp.AddAction(application.Action)

	if err := cliApp.Init(appName, appInfo, version, application.AppFlags); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize CLI app: %v\n", err)
		return
	}

	cliApp.Cli.UsageText = "so1-crawler [global options] command [command options] <url>"

	ctx := context.Background()
	args := os.Args

	if err := cliApp.Run(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
}
