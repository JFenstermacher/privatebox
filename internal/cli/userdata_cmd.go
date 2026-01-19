package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"privatebox/internal/config"
	"privatebox/internal/orchestration"
	"privatebox/internal/providers"
	"privatebox/internal/providers/aws"
	"privatebox/internal/userdata"

	"github.com/urfave/cli/v3"
)

func UserDataCmd() *cli.Command {
	return &cli.Command{
		Name:    "userdata",
		Aliases: []string{"udata"},
		Usage:   "Manage user-data scripts",
		Commands: []*cli.Command{
			{
				Name:      "create",
				Usage:     "Create a stored user-data script",
				ArgsUsage: "<name> [file_path]",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().Get(0)
					if name == "" {
						return fmt.Errorf("name is required")
					}
					filePath := cmd.Args().Get(1)

					var content []byte
					var err error

					if filePath != "" {
						content, err = os.ReadFile(filePath)
						if err != nil {
							return fmt.Errorf("failed to read file: %w", err)
						}
					} else {
						// Read from stdin
						// Check if stdin is a pipe
						stat, _ := os.Stdin.Stat()
						if (stat.Mode() & os.ModeCharDevice) != 0 {
							return fmt.Errorf("no file path provided and stdin is empty")
						}

						content, err = io.ReadAll(os.Stdin)
						if err != nil {
							return fmt.Errorf("failed to read stdin: %w", err)
						}
					}

					if len(content) == 0 {
						return fmt.Errorf("content is empty")
					}

					m, err := userdata.NewManager()
					if err != nil {
						return err
					}

					if err := m.Create(name, content); err != nil {
						return err
					}

					fmt.Printf("User-data script '%s' created.\n", name)
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List stored user-data scripts",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					m, err := userdata.NewManager()
					if err != nil {
						return err
					}
					list, err := m.List()
					if err != nil {
						return err
					}
					for _, name := range list {
						fmt.Println(name)
					}
					return nil
				},
			},
			{
				Name:      "delete",
				Usage:     "Delete a stored user-data script",
				ArgsUsage: "<name>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					name := cmd.Args().Get(0)
					if name == "" {
						return fmt.Errorf("name is required")
					}

					loader, err := config.NewLoader()
					if err != nil {
						return fmt.Errorf("failed to create config loader: %w", err)
					}
					appCfg, err := loader.Load()
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}
					cfg := appCfg.Profiles[appCfg.CurrentProfile]

					// Provider setup - assume AWS for now as per project state
					var p providers.CloudProvider = aws.NewAWSProvider(cfg)

					instances, err := orchestration.FindInstancesUsingUserData(ctx, &cfg, p, name)
					if err != nil {
						return fmt.Errorf("failed to check instances usage: %w", err)
					}

					if len(instances) > 0 {
						return fmt.Errorf("cannot delete user-data '%s': used by instances: %s", name, strings.Join(instances, ", "))
					}

					m, err := userdata.NewManager()
					if err != nil {
						return err
					}

					if err := m.Delete(name); err != nil {
						return err
					}

					fmt.Printf("User-data script '%s' deleted.\n", name)
					return nil
				},
			},
		},
	}
}
