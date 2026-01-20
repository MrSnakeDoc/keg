package brew

import (
	"context"

	"github.com/MrSnakeDoc/keg/internal/runner"
)

// BrewState represents the current state of Homebrew packages.
// This is now a wrapper around UnifiedCache for backward compatibility.
type BrewState struct {
	Installed map[string]bool
	Outdated  map[string]PackageInfo
}

// PackageInfo contains version information for a package.
type PackageInfo struct {
	Name             string
	InstalledVersion string
	LatestVersion    string
}

// brewOutdatedJSON matches the structure of `brew outdated --json=v2`.
type brewOutdatedJSON struct {
	Formulae []struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	} `json:"formulae"`
}

// FetchState retrieves the current brew state using the unified cache.
func FetchState(r runner.CommandRunner) (*BrewState, error) {
	cache, err := GetCache(r)
	if err != nil {
		return nil, err
	}

	// Refresh cache if needed (respects TTL)
	if err := cache.Refresh(context.Background(), false); err != nil {
		return nil, err
	}

	return &BrewState{
		Installed: cache.GetInstalledSet(),
		Outdated:  cache.GetOutdatedMap(),
	}, nil
}

// FetchOutdatedPackages is kept for backward compatibility.
// It now uses the unified cache internally.
func FetchOutdatedPackages(r runner.CommandRunner) (*brewOutdatedJSON, error) {
	cache, err := GetCache(r)
	if err != nil {
		return nil, err
	}

	if err := cache.Refresh(context.Background(), false); err != nil {
		return nil, err
	}

	outdatedMap := cache.GetOutdatedMap()

	result := &brewOutdatedJSON{
		Formulae: make([]struct {
			Name              string   `json:"name"`
			InstalledVersions []string `json:"installed_versions"`
			CurrentVersion    string   `json:"current_version"`
		}, 0, len(outdatedMap)),
	}

	for name, info := range outdatedMap {
		formula := struct {
			Name              string   `json:"name"`
			InstalledVersions []string `json:"installed_versions"`
			CurrentVersion    string   `json:"current_version"`
		}{
			Name:              name,
			InstalledVersions: []string{info.InstalledVersion},
			CurrentVersion:    info.LatestVersion,
		}
		result.Formulae = append(result.Formulae, formula)
	}

	return result, nil
}
