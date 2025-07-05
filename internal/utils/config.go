package utils

import (
	"fmt"
	"os"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/models"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads the package configuration from keg.yml
// It first reads the global config to get the packages file path
func LoadConfig() (*models.Config, error) {
	var config models.Config

	// First, load global config to get packages file path
	globalCfg, err := globalconfig.LoadPersistentConfig()
	if err != nil {
		return nil, err
	}

	// Read packages configuration file
	err = FileReader(globalCfg.PackagesFile, "yaml", &config)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages file %s: %w", globalCfg.PackagesFile, err)
	}

	return &config, nil
}

func SaveConfig(cfg *models.Config) error {
	fileRights := 0o644
	// Get global config for packages file path
	globalCfg, err := globalconfig.LoadPersistentConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Marshal config to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config to file
	if err := os.WriteFile(globalCfg.PackagesFile, data, os.FileMode(fileRights)); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", globalCfg.PackagesFile, err)
	}

	return nil
}
