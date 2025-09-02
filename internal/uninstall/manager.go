package uninstall

import (
	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/manifest"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Uninstall struct {
	*core.Base
}

var saveConfig = utils.SaveConfig

func New(config *models.Config, r runner.CommandRunner) *Uninstall {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Uninstall{
		Base: core.NewBase(config, r),
	}
}

func (u *Uninstall) Execute(args []string, all bool, remove bool, force bool) error {
	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:       "Uninstalling",
		ActionVerb: "uninstall",
	})

	if len(args) > 0 {
		opts.Packages = args
	}

	// Bulk mode
	if all {
		opts.FilterFunc = func(_ *models.Package) bool { return true }
	}

	// Phase 1: uninstall
	if err := u.HandlePackages(opts); err != nil {
		if !remove {
			return err
		}
	}

	if !remove {
		return nil
	}

	var toRemove []string

	if all {
		for i := range u.Config.Packages {
			p := &u.Config.Packages[i]
			toRemove = append(toRemove, p.Command)
		}
	} else {
		for _, name := range args {
			if pkg, found := u.FindPackage(name); found {
				toRemove = append(toRemove, pkg.Command)
			}
		}
	}

	if len(toRemove) == 0 {
		return nil
	}

	modified, err := manifest.RemovePackages(u.Config, toRemove)
	if err != nil {
		return err
	}
	if modified {
		if err := saveConfig(u.Config); err != nil {
			return err
		}

		logger.Success("Configuration updated successfully")
	}

	return nil
}
