package add

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Adder struct {
	*core.Base
}

func New(cfg *models.Config, r runner.CommandRunner) *Adder {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Adder{
		Base: core.NewBase(cfg, r),
	}
}

func (a *Adder) Execute(args []string, binary string, optional bool) error {
	if len(args) == 0 {
		return fmt.Errorf("no package name provided, please specify at least one package")
	}

	if err := utils.ValidateBinaryArgs(args, binary); err != nil {
		return err
	}

	modified := false
	for _, pkgName := range args {
		// Check if package already exists
		if _, found := a.FindPackage(pkgName); found {
			logger.Info("Package %s already exists in configuration", pkgName)
			continue
		}

		// Create new package entry
		newPkg := models.Package{
			Command:  pkgName,
			Optional: optional,
		}

		if binary != "" && pkgName == args[0] && binary != pkgName {
			newPkg.Binary = binary
		}

		// Add the package to config
		a.Config.Packages = append(a.Config.Packages, newPkg)
		modified = true

		logger.Success("Added %s to configuration", pkgName)
	}

	// Save config if modified
	if modified {
		if err := utils.SaveConfig(a.Config); err != nil {
			return err
		}
		logger.Success("Configuration updated successfully")
	}

	return nil
}
