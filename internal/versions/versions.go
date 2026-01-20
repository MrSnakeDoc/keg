package versions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/MrSnakeDoc/keg/internal/brew"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

// Info holds version information for a package.
// This is kept for backward compatibility with existing code.
type Info struct {
	Installed string    `json:"installed"`
	Latest    string    `json:"latest"`
	FetchedAt time.Time `json:"ts"`
}

// Resolver now wraps the unified cache for version operations.
type Resolver struct {
	Runner        runner.CommandRunner
	cache         *brew.UnifiedCache
	TTL           time.Duration
	MaxBatchSize  int
	GlobalTimeout time.Duration
	ChunkTimeout  time.Duration
}

const cacheFileName = "pkg_versions.json"

func NewResolver(r runner.CommandRunner) *Resolver {
	if r == nil {
		r = &runner.ExecRunner{}
	}

	cache, err := brew.GetCache(r)
	if err != nil {
		logger.Debug("failed to get unified cache in resolver: %v", err)
	}

	return &Resolver{
		Runner:        r,
		cache:         cache,
		TTL:           6 * time.Hour,
		MaxBatchSize:  50,
		GlobalTimeout: 15 * time.Second,
		ChunkTimeout:  5 * time.Second,
	}
}

// ResolveBulk returns version info for all names using the unified cache.
func (rv *Resolver) ResolveBulk(ctx context.Context, names []string) (map[string]Info, error) {
	if rv.cache == nil {
		logger.Debug("unified cache not available, skipping version resolution")
		return make(map[string]Info), nil
	}

	// Request version refresh for these specific packages
	if err := rv.cache.RefreshVersions(ctx, names); err != nil {
		logger.Debug("failed to refresh versions in unified cache: %v", err)
	}

	// Build result from cache
	result := make(map[string]Info, len(names))
	for _, name := range names {
		state, ok := rv.cache.GetState(name)
		if !ok {
			result[name] = Info{}
			continue
		}

		result[name] = Info{
			Installed: state.InstalledVersion,
			Latest:    state.LatestVersion,
			FetchedAt: state.FetchedAt,
		}
	}

	return result, nil
}

// Touch updates the cache for a single package after an upgrade/install.
func (rv *Resolver) Touch(name, newInstalled string) error {
	if rv.cache != nil {
		return rv.cache.MarkUpgraded(name, newInstalled)
	}
	return nil
}

// Remove deletes a package from the cache.
func (rv *Resolver) Remove(name string) error {
	if rv.cache != nil {
		return rv.cache.Invalidate(name)
	}
	return nil
}

// VersionsCachePath returns ~/.local/state/keg/pkg_versions.json (legacy, kept for compatibility).
func VersionsCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Respect XDG_STATE_HOME if present, else ~/.local/state
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(stateHome, "keg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, cacheFileName), nil
}

// LoadCache loads the legacy cache file (kept for backward compatibility).
func LoadCache() (map[string]Info, error) {
	path, err := VersionsCachePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Info{}, nil
		}
		return nil, err
	}
	var m map[string]Info
	if err := json.Unmarshal(b, &m); err != nil {
		// corrupt -> start clean
		return map[string]Info{}, nil
	}
	return m, nil
}

// SaveCache saves the legacy cache file (kept for backward compatibility).
func SaveCache(m map[string]Info) error {
	path, err := VersionsCachePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
