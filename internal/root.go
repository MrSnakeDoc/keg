package internal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/notifier"

	"github.com/spf13/cobra"
)

var noUpdate bool

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keg",
		Short: "Package installer for development environment",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "init" {
				return nil
			}

			_, err := globalconfig.LoadPersistentConfig()
			if err != nil {
				return fmt.Errorf("keg is not initialized, please run 'keg init' first: %w", err)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, _ []string) {
			versionFlag, _ := cmd.Flags().GetBool("version")
			if versionFlag {
				fmt.Printf("Version: %s\n", checker.Version)
			}
		},
	}

	cmd.Flags().BoolP("version", "v", false, "Print version information")
	cmd.PersistentFlags().BoolVar(&noUpdate, "no-update-check", false, "Skip update check")

	RegisterSubCommands(cmd)

	return cmd
}

func updateChecker(wg *sync.WaitGroup) {
	defer wg.Done()
	check := checker.New(context.Background(), nil, nil)
	if _, err := check.Execute(); err != nil {
		logger.Debug("Failed to check for updates: %v", err)
	}
}

func Execute() error {
	if os.Getenv("COMP_LINE") != "" ||
		(len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "__complete")) {
		return NewRootCmd().Execute()
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go updateChecker(&wg)

	wg.Wait()
	if err := NewRootCmd().Execute(); err != nil {
		logger.Debug("Failed to execute root command: %v", err)
		return err
	}

	if !noUpdate {
		notifier.DisplayUpdateNotification()
	}

	return nil
}
