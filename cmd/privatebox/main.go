package main

import (
	"context"
	"fmt"
	"os"

	internalCli "privatebox/internal/cli"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "privatebox",
		Usage: "Manage remote cloud instances",
		Commands: []*cli.Command{
			internalCli.ConfigCommand(),
			internalCli.InstanceCommands(),
			{
				Name:  "hello",
				Usage: "Say hello",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("Hello from Privatebox!")
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
