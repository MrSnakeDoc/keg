package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"
)

const (
	CacheDir     = ".local/state/keg"
	OutdatedFile = "keg_brew_update.json"
	CacheExpiry  = 24 * time.Hour
)

func DefaultUpdateState() map[string]interface{} {
	return map[string]interface{}{
		"last_checked":     time.Now().Add(-CacheExpiry).UTC().Format(time.RFC3339Nano),
		"update_available": false,
		"latest_version":   "",
	}
}

func EnsureUpdateStateFileExists() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Debug("failed to get user home directory: %w", err)
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	updateStateFile := filepath.Join(home, CacheDir, "update-check.json")

	if ok, _ := FileExists(updateStateFile); !ok {
		logger.Debug("update state file does not exist: %s", updateStateFile)

		defaultState := DefaultUpdateState()

		if err = CreateFile(updateStateFile, defaultState, "json", 0o644); err != nil {
			logger.Debug("failed to create update state file: %w", err)
			return "", fmt.Errorf("failed to create update state file: %w", err)
		}
		return updateStateFile, nil
	}

	return updateStateFile, nil
}
