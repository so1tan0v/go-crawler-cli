package domain

import (
	"context"

	"github.com/urfave/cli/v3"
)

type Cli interface {
	Init(appName, appInfo string, version string, flags []cli.Flag) error
	Run(ctx context.Context, args []string) error
	AddAction(func(ctx context.Context, command *cli.Command) error)
}
