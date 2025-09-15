package install

import (
	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/manifest"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

type Installer struct {
	*core.Base
}

var saveConfig = globalconfig.SaveConfig

func New(config *models.Config, r runner.CommandRunner) *Installer {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Installer{
		Base: core.NewBase(config, r),
	}
}

func (i *Installer) Execute(args []string, all bool, add bool, optional bool, binary string) error {
	// 2) Optionally update manifest first
	if add {
		// manifest.AddPackages mutates cfg in-memory
		modified, err := manifest.AddPackages(i.Config, i.FindPackage, args, binary, optional)
		if err != nil {
			return err
		}
		if modified {
			if err := saveConfig(i.Config); err != nil {
				return err
			}
			logger.Success("Configuration updated successfully")
		}
	}

	// 3) Build opts and run
	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:        "Installing",
		ActionVerb:  "install",
		SkipMessage: "%s is already installed",
	})
	if len(args) > 0 {
		opts.Packages = args
	}
	if all {
		opts.FilterFunc = func(_ *models.Package) bool { return true }
	}
	return i.HandlePackages(opts)
}
