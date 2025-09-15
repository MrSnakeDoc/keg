package globalconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/MrSnakeDoc/keg/internal/utils/pathutils"

	"gopkg.in/yaml.v3"
)

type PersistentConfig struct {
	PackagesFile string `yaml:"packages_file"`
}

const (
	configDir  = ".config/keg"
	configFile = "config.yml"

	DataDir = ".local/state/keg/gzip"

	// mounted volume in the container
	BrewFormulaURL  = "https://formulae.brew.sh/api/formula.json"
	RefreshInterval = 24 * time.Hour

	// Timeouts HTTP
	DialTimeout           = 5 * time.Second
	TLSHandshakeTimeout   = 5 * time.Second
	ResponseHeaderTimeout = 30 * time.Second

	// Global Deadline by request (incl. download gzip)
	RequestDeadline = 5 * time.Minute

	// Download security limit
	MaxDownloadBytes = 40 * 1024 * 1024 // 40 MB
)

func GetConfigDir(dirPath string) string {
	home := utils.GetHomeDir()

	return filepath.Join(home, dirPath)
}

func LoadPersistentConfig() (*PersistentConfig, error) {
	fullConfigDir := GetConfigDir(configDir)

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

	fullConfigDir := GetConfigDir(configDir)

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

func SaveConfig(cfg *models.Config) error {
	fileRights := 0o644

	globalCfg, err := LoadPersistentConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Make a shallow copy so we don't mutate in-memory order.
	cfgOut := *cfg
	cfgOut.Packages = append([]models.Package(nil), cfg.Packages...)

	// Sort: core first, optional last; alpha by Command (fallback to Binary)
	utils.SortByTypeAndKey(cfgOut.Packages,
		func(p models.Package) string {
			if p.Optional {
				return "optional"
			}
			return "core"
		},
		func(p models.Package) string {
			k := p.Command
			if k == "" {
				k = p.Binary
			}
			return strings.ToLower(k)
		},
	)

	data, err := yaml.Marshal(&cfgOut)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(globalCfg.PackagesFile, data, os.FileMode(fileRights)); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", globalCfg.PackagesFile, err)
	}

	return nil
}
