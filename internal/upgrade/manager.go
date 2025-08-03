package upgrade

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/brew"
	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type Upgrader struct {
	*core.Base
}

type packageGroups struct {
	configured []string
	deps       []string
}

func New(config *models.Config, r runner.CommandRunner) *Upgrader {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	return &Upgrader{
		Base: core.NewBase(config, r),
	}
}

func (u *Upgrader) Execute(args []string, checkOnly bool) error {
	if checkOnly {
		return u.CheckUpgrades(args)
	}

	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:       "Upgrading",
		ActionVerb: "upgrade",
	})

	if len(args) > 0 {
		opts.Packages = args
	}

	opts.FilterFunc = func(p *models.Package) bool {
		if !p.Optional {
			return true
		}
		return u.IsPackageInstalled(u.GetPackageName(p))
	}

	return u.HandlePackages(opts)
}

func (u *Upgrader) checkSinglePackage(name string, versionMap map[string]brew.PackageInfo) error {
	if !u.IsPackageInstalled(name) {
		return fmt.Errorf("package %s is not installed", name)
	}

	if v, isOutdated := versionMap[name]; isOutdated {
		p := printer.NewColorPrinter()
		utils.CreateStatusTable("", []utils.PackageStatus{{
			Name:      v.Name,
			Installed: v.InstalledVersion,
			Status:    p.Warning("outdated"),
		}})
		return nil
	}

	logger.Success("Package %s is up to date\n", name)
	return nil
}

func (u *Upgrader) sortPackages(state *brew.BrewState) packageGroups {
	configured := utils.Map(u.Config.Packages, func(p models.Package) string {
		return u.GetPackageName(&p)
	})

	configuredSet := make(map[string]bool, len(configured))
	for _, name := range configured {
		configuredSet[name] = true
	}

	installed := utils.Keys(state.Installed)

	deps := utils.Filter(installed, func(name string) bool { return !configuredSet[name] })

	return packageGroups{configured: configured, deps: deps}
}

func (u *Upgrader) checkPackageStatus(names []string, st *brew.BrewState, title string) {
	p := printer.NewColorPrinter()

	statuses := utils.Map(names, func(orig string) utils.PackageStatus {
		nameForLookup, display := orig, orig

		if pkg, found := u.FindPackage(orig); found && pkg.Optional {
			display = fmt.Sprintf("%s %s", orig, p.Warning("(opt)"))
		}

		status := utils.PackageStatus{Name: display}

		if _, ok := st.Installed[nameForLookup]; !ok {
			status.Installed = p.Error("N")
			status.Status = p.Warning("not installed")
			return status
		}

		status.Installed = p.Success("Y")
		if _, outdated := st.Outdated[nameForLookup]; outdated {
			status.Status = p.Warning("outdated")
		} else {
			status.Status = p.Success("up to date")
		}
		return status
	})

	utils.CreateStatusTable(title, statuses)
}

func (u *Upgrader) CheckUpgrades(args []string) error {
	state, err := brew.FetchState()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		pkg, found := u.FindPackage(args[0])
		if !found {
			return fmt.Errorf("package %s not found in config", args[0])
		}
		return u.checkSinglePackage(pkg.Command, state.Outdated)
	}

	groups := u.sortPackages(state)
	u.checkPackageStatus(groups.configured, state, "")
	u.checkPackageStatus(groups.deps, state, "Dependencies:")

	return nil
}
