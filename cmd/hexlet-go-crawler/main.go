package main

import (
	"code/src/domain"
	cliapp "code/src/infra/cli-app/cli-app"
	"context"
	"log"
	"os"
)

const (
	appName = "hexlet-go-crawler"
	appInfo = "analyze a website structure"
	version = "0.0.1"
)

func main() {
	cliApp := cliapp.NewCliApp()

	if err := cliApp.Init(appName, appInfo, version, domain.Flags); err != nil {
		log.Fatalf("Failed to initialize CLI app: %v", err)
	}

	ctx := context.Background()
	args := os.Args

	if err := cliApp.Run(ctx, args); err != nil {
		log.Fatalf("Failed to run CLI app: %v", err)
	}
}
