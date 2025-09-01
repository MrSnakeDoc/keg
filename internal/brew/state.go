package brew

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type BrewState struct {
	Installed map[string]string
	Outdated  map[string]PackageInfo
}

type PackageInfo struct {
	Name             string
	InstalledVersion string
	LatestVersion    string
}

type brewOutdatedJSON struct {
	Formulae []struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	} `json:"formulae"`
}

type cacheFile struct {
	Data      *brewOutdatedJSON `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
}

func readCache(filename string) (*brewOutdatedJSON, error) {
	path := utils.MakeFilePath(utils.CacheDir, filename)

	var cache cacheFile
	if err := utils.FileReader(path, "json", &cache); err != nil {
		return nil, err
	}

	// Check if cache is expired
	if time.Since(cache.Timestamp) > utils.CacheExpiry {
		return nil, fmt.Errorf("cache expired")
	}

	return cache.Data, nil
}

func FetchOutdatedPackages(r runner.CommandRunner) (*brewOutdatedJSON, error) {
	// 1. call to `brew outdated --json=v2`
	output, err := r.Run(context.Background(), 120*time.Second,
		runner.Capture, "brew", "outdated", "--json=v2")
	if err != nil {
		return nil, fmt.Errorf("failed to get outdated packages: %w", err)
	}

	// 2. Look for the first “{”
	idx := bytes.IndexByte(output, '{')
	if idx == -1 {
		return nil, fmt.Errorf("no JSON found in brew output:\n%s", output)
	}
	jsonPart := output[idx:]

	// 3. Decoding JSON
	var outdated brewOutdatedJSON
	if err := json.Unmarshal(jsonPart, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse brew JSON: %w", err)
	}

	// 4. Write the cache
	cache := cacheFile{Data: &outdated, Timestamp: time.Now()}
	if err := utils.CreateFile(
		utils.MakeFilePath(utils.CacheDir, utils.OutdatedFile),
		cache, "json", 0o600); err != nil {
		return nil, fmt.Errorf("failed to write cache: %w", err)
	}

	return &outdated, nil
}

func FetchState(r runner.CommandRunner) (*BrewState, error) {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	installed, err := utils.MapInstalledPackagesWith(r, func(pkg string) (string, string) {
		return pkg, pkg
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get installed packages: %w", err)
	}

	outdated, err := getOutdatedPackages(r)
	if err != nil {
		return nil, err
	}

	versionMap := make(map[string]PackageInfo)
	for _, f := range outdated.Formulae {
		if len(f.InstalledVersions) > 0 {
			versionMap[f.Name] = PackageInfo{
				Name:             f.Name,
				InstalledVersion: f.InstalledVersions[0],
				LatestVersion:    f.CurrentVersion,
			}
		}
	}

	return &BrewState{
		Installed: installed,
		Outdated:  versionMap,
	}, nil
}

func getOutdatedPackages(r runner.CommandRunner) (*brewOutdatedJSON, error) {
	data, err := readCache(utils.OutdatedFile)
	if err != nil {
		return FetchOutdatedPackages(r)
	}

	return data, nil
}
