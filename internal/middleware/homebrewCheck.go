package middleware

import (
	"errors"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/spf13/cobra"
)

var ErrHomebrewMissing = errors.New("homebrew is required but not installed")

func warningBrewMessages() {
	logger.Warn("Please install Homebrew first using:")
	logger.Warn("/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
	logger.Warn("Then source your shell configuration: source ~/.zshrc")
	logger.Warn("Or use the command: keg deploy")
}

func IsHomebrewInstalled(cmd *cobra.Command, args []string, next func(*cobra.Command, []string) error) error {
	if ok := utils.CommandExists("brew"); !ok {
		if cmd.Root().SilenceErrors {
			logger.LogError("%s", ErrHomebrewMissing.Error())
		}
		warningBrewMessages()
		return ErrHomebrewMissing
	}

	return next(cmd, args)
}
