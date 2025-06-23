package uninstall

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

type Uninstaller struct {
	*core.Base
}

func New(config *models.Config, r runner.CommandRunner) *Uninstaller {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Uninstaller{
		Base: core.NewBase(config, r),
	}
}

func (u *Uninstaller) Execute(args []string, all bool) error {
	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:       "Uninstalling",
		ActionVerb: "uninstall",
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

	return u.HandlePackages(opts)
}
