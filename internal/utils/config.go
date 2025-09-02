package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/models"

	"gopkg.in/yaml.v3"
)

func SaveConfig(cfg *models.Config) error {
	fileRights := 0o644

	globalCfg, err := globalconfig.LoadPersistentConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// Make a shallow copy so we don't mutate in-memory order.
	cfgOut := *cfg
	cfgOut.Packages = append([]models.Package(nil), cfg.Packages...)

	// Sort: core first, optional last; alpha by Command (fallback to Binary)
	SortByTypeAndKey(cfgOut.Packages,
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
