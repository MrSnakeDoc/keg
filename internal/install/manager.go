package install

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

type Installer struct {
	*core.Base
}

func New(config *models.Config, r runner.CommandRunner) *Installer {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Installer{
		Base: core.NewBase(config, r),
	}
}

func (i *Installer) Execute(args []string, all bool) error {
	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:        "Installing",
		ActionVerb:  "install",
		SkipMessage: "%s is already installed",
	})

	if len(args) > 0 && all {
		return fmt.Errorf("you cannot use --all with specific packages")
	}

	if len(args) > 0 {
		opts.Packages = args
	}

	if all {
		opts.FilterFunc = func(_ *models.Package) bool { return true }
	}

	return i.HandlePackages(opts)
}
