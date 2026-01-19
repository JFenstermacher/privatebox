package main

import (
	"context"
	"fmt"
	"os"

	internalCli "privatebox/internal/cli"

	"github.com/urfave/cli/v3"
)

func main() {
	commands := []*cli.Command{
		internalCli.ConfigCommand(),
		internalCli.UserDataCmd(),
	}
	commands = append(commands, internalCli.GetRootCommands()...)

	cmd := &cli.Command{
		Name:     "privatebox",
		Usage:    "Manage remote cloud instances",
		Commands: commands,
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
