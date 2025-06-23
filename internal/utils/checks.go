package utils

import (
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
)

func WarningBrewMessages() {
	logger.LogError("Homebrew is required but not installed.")
	logger.Warn("Please install Homebrew first using:")
	logger.Warn("/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
	logger.Warn("Then source your shell configuration: source ~/.zshrc")
	logger.Warn("Or use the command: plugins deploy")
}

func PreliminaryChecks() (*models.Config, error) {
	if !IsHomebrewInstalled() {
		WarningBrewMessages()
		return nil, nil
	}

	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
