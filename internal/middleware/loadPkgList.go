package middleware

import (
	"context"
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/spf13/cobra"
)

// LoadConfig loads the package configuration from keg.yml
// It first reads the global config to get the packages file path
func LoadConfig(cmd *cobra.Command) (*models.Config, error) {
	var config models.Config

	// First, load global config to get packages file path
	globalCfg, err := Get[*globalconfig.PersistentConfig](cmd, CtxKeyPConfig)
	if err != nil {
		return nil, err
	}

	// Read packages configuration file
	err = utils.FileReader(globalCfg.PackagesFile, "yaml", &config)
	if err != nil {
		return nil, fmt.Errorf("failed to read packages file %s: %w", globalCfg.PackagesFile, err)
	}

	return &config, nil
}

func LoadPkgList(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error {
	cfg, err := LoadConfig(cmd)
	if err != nil {
		return err
	}

	ctx := context.WithValue(cmd.Context(), CtxKeyConfig, cfg)
	cmd.SetContext(ctx)

	return next(cmd, args)
}
