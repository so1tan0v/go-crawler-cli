package cli_app

import (
	"context"

	"github.com/urfave/cli/v3"
)

type CliApp struct {
	Cli    cli.Command
	Action func(context.Context, *cli.Command) error
}

func NewCliApp() *CliApp {
	return &CliApp{}
}

func (c *CliApp) Init(appName, appInfo string, version string, flags []cli.Flag) error {
	c.Cli = cli.Command{
		Name:    appName,
		Version: version,
		Usage:   appInfo,
		Flags:   flags,
		Action:  c.Action,
	}

	return nil
}

func (c *CliApp) AddAction(f func(ctx context.Context, command *cli.Command) error) {
	c.Action = f
}

func (c *CliApp) Run(ctx context.Context, args []string) error {
	if err := c.Cli.Run(ctx, args); err != nil {
		return err
	}

	return nil
}
