package list

import (
	"context"
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/MrSnakeDoc/keg/internal/versions"
	"github.com/olekukonko/tablewriter"
)

type Lister struct {
	Config *models.Config
	Runner runner.CommandRunner
}

func New(config *models.Config, r runner.CommandRunner) *Lister {
	if r == nil {
		r = &runner.ExecRunner{}
	}
	return &Lister{
		Config: config,
		Runner: r,
	}
}

// Execute renders the list table.
// - onlyDeps=false => manifest only
// - onlyDeps=true  => only deps/ad-hoc (installed but not in manifest)
func (l *Lister) Execute(ctx context.Context, onlyDeps bool) error {
	// 1) installed
	installed, err := utils.MapInstalledPackagesWith(l.Runner, func(pkg string) (string, bool) {
		return pkg, true
	})
	if err != nil {
		return fmt.Errorf("an error occurred while fetching installed packages: %w", err)
	}

	// 2) configured + sets
	configured, cfgSet, optionalSet := l.buildConfigured()

	// 3) deps
	deps := l.computeDeps(installed, cfgSet)

	// 4) choose names
	names := configured
	if onlyDeps {
		names = deps
	}

	// 5) resolve versions
	resolver := versions.NewResolver(l.Runner)
	versionInfo, err := resolver.ResolveBulk(ctx, names)
	if err != nil {
		logger.Debug("version resolution failed (list): %v", err)
		versionInfo = map[string]versions.Info{}
	}

	// 6) render
	p := printer.NewColorPrinter()
	table := logger.CreateTable([]string{"Package", "Version", "Status", "Type"})

	for _, name := range names {
		status := p.Success("✓ installed")
		if !installed[name] {
			status = p.Error("✗ missing")
		}

		ver := "—"
		if vi, ok := versionInfo[name]; ok && vi.Installed != "" {
			ver = vi.Installed
		}

		pkgType := "dep"
		if !onlyDeps {
			if optionalSet[name] {
				pkgType = p.Warning("optional")
			} else if _, ok := cfgSet[name]; ok {
				pkgType = "default"
			}
		} else {
			pkgType = p.Warning("dep")
		}

		if err := renderRow(table, name, ver, status, pkgType); err != nil {
			return fmt.Errorf("an error occurred while appending to the table: %w", err)
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("an error occurred while rendering the table: %w", err)
	}

	return nil
}

func (l *Lister) buildConfigured() (names []string, cfgSet map[string]struct{}, optionalSet map[string]bool) {
	names = utils.Map(l.Config.Packages, func(p models.Package) string {
		if p.Binary != "" {
			return p.Binary
		}
		return p.Command
	})
	cfgSet = make(map[string]struct{}, len(names))
	optionalSet = make(map[string]bool, len(names))
	for _, p := range l.Config.Packages {
		name := p.Command
		if p.Binary != "" {
			name = p.Binary
		}
		cfgSet[name] = struct{}{}
		if p.Optional {
			optionalSet[name] = true
		}
	}
	return
}

func (l *Lister) computeDeps(installed map[string]bool, cfgSet map[string]struct{}) []string {
	return utils.Filter(utils.Keys(installed), func(n string) bool {
		_, ok := cfgSet[n]
		return !ok
	})
}

func renderRow(table *tablewriter.Table, name, ver, status, pkgType string) error {
	return table.Append([]string{name, ver, status, pkgType})
}
