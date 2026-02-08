package cli_app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestCliAppInit(t *testing.T) {
	app := NewCliApp()

	called := false
	app.AddAction(func(ctx context.Context, cmd *cli.Command) error {
		called = true
		return nil
	})

	err := app.Init("gendiff", "Compares two configuration files and shows a difference.", "0.0.1", []cli.Flag{
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage:   "output format",
			Value:   "stylish",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, called, false)
}

func TestCliAppAddAction(t *testing.T) {
	app := NewCliApp()
	var receivedCtx context.Context
	var receivedCmd *cli.Command

	app.AddAction(func(ctx context.Context, cmd *cli.Command) error {
		receivedCtx = ctx
		receivedCmd = cmd

		return nil
	})

	assert.NotNil(t, app.Action)

	ctx := context.Background()
	dummyCmd := &cli.Command{}
	err := app.Action(ctx, dummyCmd)

	require.NoError(t, err)
	assert.Equal(t, ctx, receivedCtx)
	assert.Equal(t, dummyCmd, receivedCmd)
}

func TestCliAppRunSuccess(t *testing.T) {
	app := NewCliApp()

	formatValue := ""
	called := false

	app.AddAction(func(ctx context.Context, cmd *cli.Command) error {
		called = true
		formatValue = cmd.String("format")
		return nil
	})

	err := app.Init()
	require.NoError(t, err)

	ctx := context.Background()
	args := []string{"gendiff", "--format=plain"}

	err = app.Run(ctx, args)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "plain", formatValue)
}

func TestCliAppRunWithAlias(t *testing.T) {
	app := NewCliApp()

	formatValue := ""
	app.AddAction(func(_ context.Context, cmd *cli.Command) error {
		formatValue = cmd.String("format")
		return nil
	})

	err := app.Init()
	require.NoError(t, err)

	err = app.Run(context.Background(), []string{"gendiff", "-f", "json"})
	require.NoError(t, err)
	assert.Equal(t, "json", formatValue)
}
