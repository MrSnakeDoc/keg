package globalconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MrSnakeDoc/keg/internal/utils/pathutils"

	"gopkg.in/yaml.v3"
)

type PersistentConfig struct {
	PackagesFile string `yaml:"packages_file"`
}

const (
	configDir  = ".config/keg"
	configFile = "config.yml"
)

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, configDir), nil
}

func LoadPersistentConfig() (*PersistentConfig, error) {
	fullConfigDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(fullConfigDir, configFile)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no configuration found. Please run 'keg init' first")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg PersistentConfig
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	absPath, err := pathutils.ToAbsolutePath(cfg.PackagesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve packages file path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("packages file not found at %s: %w", cfg.PackagesFile, err)
	}

	cfg.PackagesFile = absPath
	return &cfg, nil
}

func (c *PersistentConfig) Save() error {
	configDirRights := 0o755
	configFileRights := 0o644

	fullConfigDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(fullConfigDir, os.FileMode(configDirRights)); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	homePath, err := pathutils.ToHomePathFormat(c.PackagesFile)
	if err != nil {
		return fmt.Errorf("failed to convert to home path format: %w", err)
	}
	c.PackagesFile = homePath

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filepath.Join(fullConfigDir, configFile), data, os.FileMode(configFileRights))
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
