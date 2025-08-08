package middleware

import (
	"fmt"
	"os/exec"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/spf13/cobra"
)

func WarningBrewMessages() {
	logger.LogError("Homebrew is required but not installed.")
	logger.Warn("Please install Homebrew first using:")
	logger.Warn("/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
	logger.Warn("Then source your shell configuration: source ~/.zshrc")
	logger.Warn("Or use the command: keg deploy")
}

func IsHomebrewInstalled(cmd *cobra.Command, args []string, next func(*cobra.Command, []string) error) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf(
			"homebrew not found in PATH: %w\n"+
				"hint: install it with `/bin/bash -c \"$(curl -fsSL "+
				"https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"` or run `keg deploy`",
			err,
		)
	}
	return next(cmd, args)
}
