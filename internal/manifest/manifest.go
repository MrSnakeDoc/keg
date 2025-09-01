package manifest

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

// Finder abstracts how to find a package in the current config.
type Finder func(name string) (*models.Package, bool)

// AddPackages mutates cfg.Packages by appending new packages if absent.
// Returns true if cfg was modified.
func AddPackages(cfg *models.Config, find Finder, names []string, binary string, optional bool) (bool, error) {
	if len(names) == 0 {
		return false, fmt.Errorf("no package name provided, please specify at least one package")
	}
	if err := utils.ValidateBinaryArgs(names, binary); err != nil {
		return false, err
	}

	modified := false
	for idx, name := range names {
		if _, exists := find(name); exists {
			// Package already present, skip silently (idempotent)
			continue
		}

		pkg := models.Package{
			Command:  name,
			Optional: optional,
		}
		// Only first package can take the --binary (same rule as before)
		if binary != "" && idx == 0 && binary != name {
			pkg.Binary = binary
		}

		cfg.Packages = append(cfg.Packages, pkg)
		modified = true
	}
	return modified, nil
}

// RemovePackages removes packages by name from cfg.Packages.
// Returns true if cfg was modified.
func RemovePackages(cfg *models.Config, names []string) (bool, error) {
	if len(names) == 0 {
		return false, fmt.Errorf("no package name provided, please specify at least one package")
	}

	nameSet := utils.TransformToMap(names, func(s string) (string, struct{}) {
		return s, struct{}{}
	})

	out := cfg.Packages[:0]
	removed := false

	for _, p := range cfg.Packages {
		if _, hit := nameSet[p.Command]; hit {
			removed = true
			continue
		}
		out = append(out, p)
	}

	if removed {
		cfg.Packages = out
	}
	return removed, nil
}
