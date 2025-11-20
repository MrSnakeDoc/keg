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
)

// row is a view model for rendering.
type row struct {
	DisplayName string // what we show in the table (binary or command as today)
	Version     string
	StatusCode  string
	Type        string // "core" | "dep" | "optional"
	SortKey     string // ALWAYS the command name for sorting
}

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
	installed, err := utils.InstalledSet(l.Runner)
	if err != nil {
		return fmt.Errorf("fetch installed packages: %w", err)
	}

	// configured names + sets + map
	configured, cfgSet, optionalSet, nameToCommand := l.buildConfigured()
	deps := l.computeDeps(installed, cfgSet)

	// choose list
	names := configured
	if onlyDeps {
		names = deps
	}

	// versions
	resolver := versions.NewResolver(l.Runner)
	versionInfo, err := resolver.ResolveBulk(ctx, names)
	if err != nil {
		logger.Debug("version resolution failed (list): %v", err)
		versionInfo = map[string]versions.Info{}
	}

	// Build rows
	rows := utils.Map(names, func(name string) row {
		status := "installed"
		if !installed[name] {
			status = "missing"
		}

		ver := "—"
		if vi, ok := versionInfo[name]; ok && vi.Installed != "" {
			ver = vi.Installed
		}

		pkgType := "dep"
		if !onlyDeps {
			if optionalSet[name] {
				pkgType = "optional"
			} else if _, ok := cfgSet[name]; ok {
				pkgType = "core"
			}
		}

		// SortKey = command name when we know it; fallback to name
		sortKey := name
		if cmd, ok := nameToCommand[name]; ok && cmd != "" {
			sortKey = cmd
		}

		return row{
			DisplayName: name,
			Version:     ver,
			StatusCode:  status,
			Type:        pkgType,
			SortKey:     sortKey,
		}
	})

	// Sort rows: core < dep < optional, then alpha by command
	utils.SortByTypeAndKey(rows, func(r row) string { return r.Type }, func(r row) string { return r.SortKey })

	return outputItems(rows)
}

func outputItems(rows []row) error {
	p := printer.NewColorPrinter()
	table := logger.CreateTable([]string{"Package", "Version", "Status", "Type"})

	for _, r := range rows {
		status := "-"
		switch r.StatusCode {
		case "installed":
			status = p.Success("installed")
		case "missing":
			status = p.Warning("not installed")
		}

		if err := logger.RenderRow(table, r.DisplayName, r.Version, status, prettyType(p, r.Type)); err != nil {
			return fmt.Errorf("append to table: %w", err)
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("render table: %w", err)
	}
	return nil
}

func (l *Lister) buildConfigured() (names []string, cfgSet map[string]struct{}, optionalSet map[string]bool, nameToCommand map[string]string) {
	names = utils.Map(l.Config.Packages, func(p models.Package) string {
		if p.Binary != "" {
			return p.Binary
		}
		return p.Command
	})

	cfgSet = make(map[string]struct{}, len(names))
	optionalSet = make(map[string]bool, len(names))
	nameToCommand = make(map[string]string, len(names))

	for _, p := range l.Config.Packages {
		name := p.Command
		if p.Binary != "" {
			name = p.Binary
		}
		cfgSet[name] = struct{}{}
		nameToCommand[name] = p.Command // <- ALWAYS store the command for sorting
		if p.Optional {
			optionalSet[name] = true
		}
	}
	return names, cfgSet, optionalSet, nameToCommand
}

func (l *Lister) computeDeps(installed map[string]bool, cfgSet map[string]struct{}) []string {
	return utils.Filter(utils.Keys(installed), func(n string) bool {
		_, ok := cfgSet[n]
		return !ok
	})
}

// prettyType colors only the UI label, not the sorting value.
func prettyType(p *printer.ColorPrinter, t string) string {
	switch t {
	case "optional":
		return p.Warning("optional")
	case "core":
		return "core"
	case "dep":
		return "dep"
	default:
		return t
	}
}
