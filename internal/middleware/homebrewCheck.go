package middleware

import (
	"os"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/spf13/cobra"
)

func WarningBrewMessages() {
	logger.LogError("Homebrew is required but not installed.")
	logger.Warn("Please install Homebrew first using:")
	logger.Warn("/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
	logger.Warn("Then source your shell configuration: source ~/.zshrc")
	logger.Warn("Or use the command: keg deploy")
}

func IsHomebrewInstalled(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error {
	if !utils.IsHomebrewInstalled() {
		WarningBrewMessages()
		os.Exit(1)
	}

	return next(cmd, args)
}
