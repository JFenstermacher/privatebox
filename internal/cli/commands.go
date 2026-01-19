package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"privatebox/internal/config"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
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
					fmt.Printf("Current Profile: %s\n", cfg.CurrentProfile)
					fmt.Println("---")

					data, _ := yaml.Marshal(cfg)
					fmt.Println(string(data))
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List all profiles",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					loader, err := config.NewLoader()
					if err != nil {
						return err
					}
					cfg, err := loader.Load()
					if err != nil {
						return err
					}

					for name := range cfg.Profiles {
						prefix := " "
						if name == cfg.CurrentProfile {
							prefix = "*"
						}
						fmt.Printf("%s %s\n", prefix, name)
					}
					return nil
				},
			},
			{
				Name:      "use",
				Usage:     "Switch current profile",
				ArgsUsage: "<profile-name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().First()
					if name == "" {
						return fmt.Errorf("profile name required")
					}

					loader, err := config.NewLoader()
					if err != nil {
						return err
					}
					cfg, err := loader.Load()
					if err != nil {
						return err
					}

					if _, ok := cfg.Profiles[name]; !ok {
						return fmt.Errorf("profile '%s' does not exist", name)
					}

					cfg.CurrentProfile = name
					if err := loader.Save(cfg); err != nil {
						return err
					}
					fmt.Printf("Switched to profile '%s'\n", name)
					return nil
				},
			},
			{
				Name:      "new",
				Usage:     "Create a new profile (clones default)",
				ArgsUsage: "<profile-name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().First()
					if name == "" {
						return fmt.Errorf("profile name required")
					}

					loader, err := config.NewLoader()
					if err != nil {
						return err
					}
					cfg, err := loader.Load()
					if err != nil {
						return err
					}

					if _, ok := cfg.Profiles[name]; ok {
						return fmt.Errorf("profile '%s' already exists", name)
					}

					// Clone default or create new default
					cfg.Profiles[name] = config.DefaultProfile()

					// If this is the first profile, set it as current
					if cfg.CurrentProfile == "" {
						cfg.CurrentProfile = name
					}

					if err := loader.Save(cfg); err != nil {
						return err
					}
					fmt.Printf("Created profile '%s'\n", name)
					return nil
				},
			},
			{
				Name:  "edit",
				Usage: "Open config file in editor",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					loader, err := config.NewLoader()
					if err != nil {
						return err
					}

					// Ensure config exists (migrate from JSON if needed) before opening
					path := loader.GetConfigPath()
					if _, err := os.Stat(path); os.IsNotExist(err) {
						cfg, err := loader.Load()
						if err != nil {
							return err
						}
						if err := loader.Save(cfg); err != nil {
							return err
						}
					}

					editor := os.Getenv("EDITOR")
					if editor == "" {
						editor = "vi"
					}

					c := exec.Command(editor, loader.GetConfigPath())
					c.Stdin = os.Stdin
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					return c.Run()
				},
			},
		},
	}
}
