package internal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/notifier"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keg",
		Short: "Package installer for development environment",
		Long: `Keg is a package installer designed to simplify the setup and management of development environments.
It allows you to easily install, update, and manage packages and dependencies for your projects.`,
		Example: `keg install lazygit asdf`,
		Run: func(cmd *cobra.Command, _ []string) {
			versionFlag, _ := cmd.Flags().GetBool("version")
			if versionFlag {
				fmt.Printf("Version: %s\n", checker.Version)
			}
		},
		PersistentPostRunE: func(cmd *cobra.Command, _ []string) error {
			noUpdate, _ := cmd.Flags().GetBool("no-update-check")

			envNoUpdate := strings.TrimSpace(os.Getenv("KEG_NO_UPDATE_CHECK")) == "1"

			v, _ := cmd.Flags().GetBool("version")

			name := cmd.Name()

			switch {
			case name == "update",
				name == "help",
				name == "completion",
				name == "keg" && v,
				envNoUpdate || noUpdate:
				return nil
			}

			check := checker.New(nil, nil)
			if _, err := check.Execute(context.Background(), false); err != nil {
				logger.Debug("Failed to check for updates: %v", err)
				return nil
			}

			notifier.DisplayUpdateNotification()

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	cmd.Flags().BoolP("version", "v", false, "Print version information")
	cmd.PersistentFlags().Bool("no-update-check", false, "Skip update check")

	RegisterSubCommands(cmd)

	return cmd
}

func Execute() error {
	root := NewRootCmd()

	if os.Getenv("COMP_LINE") != "" ||
		(len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "__complete")) {
		return root.Execute()
	}

	if err := root.Execute(); err != nil {
		logger.Debug("Failed to execute root command: %v", err)
		return err
	}
	return nil
}
