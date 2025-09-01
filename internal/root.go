package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/notifier"
	"github.com/MrSnakeDoc/keg/internal/utils"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keg",
		Short: "Package installer for development environment",
		Long: `Keg is a package installer designed to simplify the setup and management of development environments.
It allows you to easily install, update, and manage packages and dependencies for your projects.`,
		Example:       `keg install lazygit asdf`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			logger.ConfigureLoggerFromFlags()
			return nil
		},
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

			if !utils.IsSemver(checker.Version) {
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
	}

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		_, err = fmt.Fprintf(c.ErrOrStderr(), "Error: %v\n\n", err)
		if err != nil {
			return err
		}
		_ = c.Usage()
		return err
	})

	cmd.Flags().BoolP("version", "v", false, "Print version information")
	cmd.PersistentFlags().Bool("no-update-check", false, "Skip update check")
	cmd.PersistentFlags().CountVarP(&logger.FlagVerboseCount, "verbose", "V", "Increase verbosity (-V, -VV, -VVV)")
	cmd.PersistentFlags().BoolVarP(&logger.FlagSilent, "silent", "s", false, "Silent mode (no output even errors)")
	cmd.PersistentFlags().BoolVarP(&logger.FlagQuiet, "quiet", "q", false, "Quiet mode (no log output except errors)")
	cmd.PersistentFlags().BoolVarP(&logger.FlagJSON, "log-json", "j", false, "Log in JSON (no colors)")

	RegisterSubCommands(cmd)

	return cmd
}

func Execute() error {
	root := NewRootCmd()

	if strings.HasPrefix(strings.Join(os.Args, " "), "__complete") {
		logger.SetOutput(io.Discard)
	}

	if err := root.Execute(); err != nil {
		if errors.Is(err, middleware.ErrLogged) {
			os.Exit(1)
		}
		return err
	}
	return nil
}
