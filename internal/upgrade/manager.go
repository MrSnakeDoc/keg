package upgrade

import (
	"context"
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/brew"
	"github.com/MrSnakeDoc/keg/internal/core"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/MrSnakeDoc/keg/internal/versions"
)

type Upgrader struct {
	*core.Base
}

func New(config *models.Config, r runner.CommandRunner) *Upgrader {
	if r == nil {
		r = &runner.ExecRunner{}
	}
	return &Upgrader{Base: core.NewBase(config, r)}
}

func (u *Upgrader) Execute(args []string, checkOnly bool, all bool) error {
	if checkOnly {
		return u.CheckUpgrades(args, all)
	}

	opts := core.DefaultPackageHandlerOptions(core.PackageAction{
		Name:       "Upgrading",
		ActionVerb: "upgrade",
	})
	opts.AllowAdHoc = all || len(args) > 0

	if len(args) > 0 {
		opts.Packages = args
	} else if all {
		configured := make([]string, 0, len(u.Config.Packages))
		configuredSet := make(map[string]struct{}, len(u.Config.Packages))
		for i := range u.Config.Packages {
			name := u.GetPackageName(&u.Config.Packages[i])
			configured = append(configured, name)
			configuredSet[name] = struct{}{}
		}

		st, err := brew.FetchState(u.Runner)
		if err != nil {
			return err
		}
		deps := make([]string, 0, len(st.Installed))
		for name := range st.Installed {
			if _, ok := configuredSet[name]; !ok {
				deps = append(deps, name)
			}
		}
		combined := append([]string{}, configured...)
		combined = append(combined, deps...)
		opts.Packages = combined
	}

	opts.FilterFunc = func(p *models.Package) bool {
		if !p.Optional {
			return true
		}
		return u.IsPackageInstalled(u.GetPackageName(p))
	}

	return u.HandlePackages(opts)
}

/* ===========================
   Helpers for --check output
   =========================== */

func (u *Upgrader) buildConfiguredSets() (configured []string, cfgSet map[string]struct{}, optionalSet map[string]bool) {
	configured = utils.Map(u.Config.Packages, func(p models.Package) string {
		return u.GetPackageName(&p)
	})
	cfgSet = make(map[string]struct{}, len(configured))
	optionalSet = make(map[string]bool, len(configured))
	for _, p := range u.Config.Packages {
		name := u.GetPackageName(&p)
		cfgSet[name] = struct{}{}
		if p.Optional {
			optionalSet[name] = true
		}
	}
	return
}

func computeDeps(st *brew.BrewState, cfgSet map[string]struct{}) []string {
	installed := utils.Keys(st.Installed)
	return utils.Filter(installed, func(name string) bool {
		_, ok := cfgSet[name]
		return !ok
	})
}

func (u *Upgrader) normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, raw := range args {
		if pkg, found := u.FindPackage(raw); found {
			out = append(out, u.GetPackageName(pkg))
		} else {
			out = append(out, raw)
		}
	}
	return out
}

func (u *Upgrader) resolveVersions(names []string) map[string]versions.Info {
	if len(names) == 0 {
		return map[string]versions.Info{}
	}
	resolver := versions.NewResolver(u.Runner)
	vi, err := resolver.ResolveBulk(context.Background(), names)
	if err != nil {
		logger.Debug("version resolution failed (upgrade --check): %v", err)
		return map[string]versions.Info{}
	}
	return vi
}

func (u *Upgrader) renderCheckTable(title string, names []string, st *brew.BrewState, cfgSet map[string]struct{}, optionalSet map[string]bool, vers map[string]versions.Info) error {
	if len(names) == 0 {
		return nil
	}

	if title != "" {
		logger.Info(title)
	}

	p := printer.NewColorPrinter()
	table := logger.CreateTable([]string{"Package", "Version", "Status", "Type"})

	for _, name := range names {
		// status & versions
		var versionCell, statusCell, typeCell string

		if _, ok := st.Installed[name]; !ok {
			versionCell = "—"
			statusCell = p.Warning("not installed")
		} else if v, out := st.Outdated[name]; out {
			// outdated: installed -> latest
			oldV := p.Error(v.InstalledVersion)
			newV := p.Success(v.LatestVersion)
			versionCell = fmt.Sprintf("%s -> %s", oldV, newV)
			statusCell = p.Warning("outdated")
		} else {
			// up to date: show installed version in green (fallback to cache resolver)
			if info, ok := vers[name]; ok && info.Installed != "" {
				versionCell = p.Success(info.Installed)
			} else {
				versionCell = p.Success("current")
			}
			statusCell = p.Success("up to date")
		}

		// type
		if _, ok := cfgSet[name]; ok {
			if optionalSet[name] {
				typeCell = p.Warning("optional")
			} else {
				typeCell = "default"
			}
		} else {
			typeCell = p.Warning("dep")
		}

		if err := table.Append([]string{name, versionCell, statusCell, typeCell}); err != nil {
			return fmt.Errorf("an error occurred while appending to the table: %w", err)
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("an error occurred while rendering the table: %w", err)
	}
	return nil
}

/* ===========================
   CheckUpgrades (with helpers)
   =========================== */

func (u *Upgrader) CheckUpgrades(args []string, all bool) error {
	state, err := brew.FetchState(u.Runner)
	if err != nil {
		return err
	}

	configured, cfgSet, optionalSet := u.buildConfiguredSets()
	deps := computeDeps(state, cfgSet)

	// selection
	if len(args) > 0 {
		names := u.normalizeArgs(args)
		vers := u.resolveVersions(names)
		return u.renderCheckTable("", names, state, cfgSet, optionalSet, vers)
	}

	// manifest table
	versManifest := u.resolveVersions(configured)
	if err := u.renderCheckTable("", configured, state, cfgSet, optionalSet, versManifest); err != nil {
		return err
	}

	// deps table when --all
	if all && len(deps) > 0 {
		versDeps := u.resolveVersions(deps)
		if err := u.renderCheckTable("Dependencies:", deps, state, cfgSet, optionalSet, versDeps); err != nil {
			return err
		}
	}

	return nil
}
