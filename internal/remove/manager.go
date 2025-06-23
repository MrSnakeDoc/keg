package remove

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Remover struct {
	*core.Base
}

func New(cfg *models.Config, r runner.CommandRunner) *Remover {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Remover{
		Base: core.NewBase(cfg, r),
	}
}

// RemovePackagesByName removes packages from the given slice by their names.
func RemovePackagesByName(packages []models.Package, names []string) ([]models.Package, bool) {
	nameSet := utils.TransformToMap(names, func(s string) (string, struct{}) {
		return s, struct{}{}
	})

	var result []models.Package
	removed := false

	for _, pkg := range packages {
		if _, found := nameSet[pkg.Command]; found {
			removed = true
			continue
		}
		result = append(result, pkg)
	}
	return result, removed
}

func (r *Remover) Execute(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no package name provided, please specify at least one package")
	}

	existing := make(map[string]bool)
	for _, pkg := range r.Config.Packages {
		existing[pkg.Command] = true
	}

	for _, name := range args {
		if !existing[name] {
			logger.Info("Package %s doesn't exist in configuration", name)
		} else {
			logger.Success("Removed %s from configuration", name)
		}
	}

	newPackages, modified := RemovePackagesByName(r.Config.Packages, args)
	if !modified {
		return nil
	}

	r.Config.Packages = newPackages

	if err := utils.SaveConfig(r.Config); err != nil {
		return err
	}

	logger.Success("Configuration updated successfully")
	return nil
}
