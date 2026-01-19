package cli

import (
	"context"
	"fmt"
	"privatebox/internal/config"

	"github.com/urfave/cli/v3"
)

// ConfigCommand returns the CLI command for managing configuration.
func ConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage configuration",
		Commands: []*cli.Command{
			{
				Name:  "show",
				Usage: "Display current configuration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					loader, err := config.NewLoader()
					if err != nil {
						return err
					}

					cfg, err := loader.Load()
					if err != nil {
						return err
					}

					fmt.Printf("Config File: %s\n", loader.GetConfigPath())
					fmt.Println("---")

					// Re-marshal to show pretty JSON
					if err := loader.Save(cfg); err != nil {
						// If save fails (permissions?), just dump struct
						fmt.Printf("%+v\n", cfg)
					} else {
						// Read it back to print raw json or just print manual
						// A simple way is to use the save logic to print to stdout
						// But for now let's just use the file content or reconstruct
						// reusing Save logic printed to stdout:
						importConfig, _ := config.NewLoader()
						cfg, _ := importConfig.Load()
						// This is a bit circular, let's just print the struct fields for now for simplicity in this step
						fmt.Printf("Provider: %s\n", cfg.Provider)
						fmt.Printf("Region: %s\n", cfg.Region)
						fmt.Printf("Pulumi Backend: %s\n", cfg.PulumiBackend)
						fmt.Printf("AWS Instance Type: %s\n", cfg.AWS.InstanceType)
					}
					return nil
				},
			},
			{
				Name:  "init",
				Usage: "Initialize default configuration",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					loader, err := config.NewLoader()
					if err != nil {
						return err
					}

					cfg := config.DefaultConfig()
					if err := loader.Save(&cfg); err != nil {
						return err
					}

					fmt.Printf("Initialized default config at %s\n", loader.GetConfigPath())
					return nil
				},
			},
		},
	}
}
