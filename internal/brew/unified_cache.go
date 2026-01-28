package brew

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

// UnifiedCache is the single source of truth for all brew package states.
// It consolidates:
// - Installed packages (previously core.Base.installedPkgs)
// - Version information (previously versions cache)
// - Outdated status (previously brew/state.go cache)
type UnifiedCache struct {
	mu          sync.RWMutex
	LastUpdated time.Time               `json:"last_updated"`
	Packages    map[string]PackageState `json:"packages"`
	TTL         time.Duration           `json:"-"`
	runner      runner.CommandRunner    `json:"-"`
	cachePath   string                  `json:"-"`
}

// PackageState represents the complete state of a single package.
type PackageState struct {
	Installed        bool      `json:"installed"`
	InstalledVersion string    `json:"installed_version,omitempty"`
	LatestVersion    string    `json:"latest_version,omitempty"`
	Outdated         bool      `json:"outdated,omitempty"`
	FetchedAt        time.Time `json:"fetched_at"`
}

const (
	defaultTTL    = 1 * time.Hour
	cacheFileName = "unified_cache.json"
	maxBatchSize  = 50
	brewTimeout   = 120 * time.Second
)

var (
	globalCache   *UnifiedCache
	globalCacheMu sync.Mutex
)

// GetCache returns the global unified cache instance (singleton).
func GetCache(r runner.CommandRunner) (*UnifiedCache, error) {
	globalCacheMu.Lock()
	defer globalCacheMu.Unlock()

	if globalCache != nil {
		return globalCache, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}

	dir := filepath.Join(stateHome, "keg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := filepath.Join(dir, cacheFileName)

	if r == nil {
		r = &runner.ExecRunner{}
	}

	cache := &UnifiedCache{
		Packages:  make(map[string]PackageState),
		TTL:       defaultTTL,
		runner:    r,
		cachePath: cachePath,
	}

	// Try to load existing cache from disk
	_ = cache.loadFromDisk()

	globalCache = cache
	return globalCache, nil
}

// ResetCache clears the global cache (useful for testing).
func ResetCache() {
	globalCacheMu.Lock()
	defer globalCacheMu.Unlock()
	globalCache = nil
}

// IsInstalled checks if a package is installed.
func (c *UnifiedCache) IsInstalled(name string) bool {
	c.mu.RLock()
	needsRefresh := c.needsRefreshLocked()
	c.mu.RUnlock()

	if needsRefresh {
		_ = c.Refresh(context.Background(), false)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	pkg, ok := c.Packages[name]
	return ok && pkg.Installed
}

// GetState returns the complete state for a package.
func (c *UnifiedCache) GetState(name string) (PackageState, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pkg, ok := c.Packages[name]
	return pkg, ok
}

// GetInstalledSet returns a map of all installed packages.
func (c *UnifiedCache) GetInstalledSet() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]bool, len(c.Packages))
	for name, pkg := range c.Packages {
		if pkg.Installed {
			result[name] = true
		}
	}
	return result
}

// GetOutdatedMap returns packages that need updating.
func (c *UnifiedCache) GetOutdatedMap() map[string]PackageInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]PackageInfo)
	for name, pkg := range c.Packages {
		if pkg.Outdated && pkg.Installed {
			result[name] = PackageInfo{
				Name:             name,
				InstalledVersion: pkg.InstalledVersion,
				LatestVersion:    pkg.LatestVersion,
			}
		}
	}
	return result
}

// Refresh updates the cache by calling brew commands.
// If force=true, ignores TTL and always refreshes.
func (c *UnifiedCache) Refresh(ctx context.Context, force bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !force && !c.needsRefreshLocked() {
		logger.Debug("unified cache is fresh, skipping refresh (age: %s)", time.Since(c.LastUpdated).Truncate(time.Second))
		return nil
	}

	logger.Debug("refreshing unified cache...")
	start := time.Now()

	// Step 1: Get installed packages (brew list)
	installed, err := c.fetchInstalledPackages()
	if err != nil {
		return fmt.Errorf("failed to fetch installed packages: %w", err)
	}

	// Step 2: Get outdated packages (brew outdated --json=v2)
	outdated, err := c.fetchOutdatedPackages()
	if err != nil {
		logger.Debug("failed to fetch outdated packages (non-fatal): %v", err)
		outdated = &brewOutdatedJSON{}
	}

	// Step 3: Build outdated map
	outdatedMap := make(map[string]PackageInfo)
	for _, f := range outdated.Formulae {
		if len(f.InstalledVersions) > 0 {
			outdatedMap[f.Name] = PackageInfo{
				Name:             f.Name,
				InstalledVersion: f.InstalledVersions[0],
				LatestVersion:    f.CurrentVersion,
			}
		}
	}

	// Step 4: Update cache
	now := time.Now()
	newPackages := make(map[string]PackageState, len(installed))

	for name := range installed {
		state := PackageState{
			Installed: true,
			FetchedAt: now,
		}

		// Check if outdated
		if info, isOutdated := outdatedMap[name]; isOutdated {
			state.Outdated = true
			state.InstalledVersion = info.InstalledVersion
			state.LatestVersion = info.LatestVersion
		} else {
			state.Outdated = false
			// Keep existing version info if available
			if old, ok := c.Packages[name]; ok {
				state.InstalledVersion = old.InstalledVersion
				state.LatestVersion = old.LatestVersion
			}
		}

		newPackages[name] = state
	}

	c.Packages = newPackages
	c.LastUpdated = now

	logger.Debug("unified cache refreshed in %s (%d packages)", time.Since(start).Truncate(time.Millisecond), len(c.Packages))

	// Save to disk
	return c.saveToDiskLocked()
}

// RefreshVersions fetches version info for specific packages using brew info.
func (c *UnifiedCache) RefreshVersions(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Dedupe
	seen := make(map[string]struct{}, len(names))
	unique := make([]string, 0, len(names))
	for _, n := range names {
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			unique = append(unique, n)
		}
	}

	// Batch into chunks
	chunks := chunkStrings(unique, maxBatchSize)
	now := time.Now()

	for _, chunk := range chunks {
		versionInfo, err := c.fetchVersionsForChunk(ctx, chunk)
		if err != nil {
			logger.Debug("failed to fetch versions for chunk: %v", err)
			continue
		}

		// Update cache with version info
		for name, info := range versionInfo {
			state, ok := c.Packages[name]
			if !ok {
				// Package not in cache yet, create entry
				state = PackageState{
					FetchedAt: now,
				}
			}

			state.InstalledVersion = info.InstalledVersion
			state.LatestVersion = info.LatestVersion
			state.FetchedAt = now

			// CRITICAL: Update installed flag based on version info
			state.Installed = (info.InstalledVersion != "")

			if state.InstalledVersion != "" && state.LatestVersion != "" {
				state.Outdated = state.InstalledVersion != state.LatestVersion
			}

			c.Packages[name] = state
		}
	}

	return c.saveToDiskLocked()
}

// Invalidate marks specific packages for refresh.
func (c *UnifiedCache) Invalidate(names ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(names) == 0 {
		// Invalidate all
		c.LastUpdated = time.Time{}
		return c.saveToDiskLocked()
	}

	// Invalidate specific packages
	for _, name := range names {
		delete(c.Packages, name)
	}

	return c.saveToDiskLocked()
}

// MarkInstalled updates the cache after a package installation.
func (c *UnifiedCache) MarkInstalled(name, version string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.Packages[name]
	state.Installed = true
	state.InstalledVersion = version
	state.LatestVersion = version
	state.Outdated = false
	state.FetchedAt = time.Now()

	c.Packages[name] = state

	return c.saveToDiskLocked()
}

// MarkUninstalled updates the cache after a package removal.
func (c *UnifiedCache) MarkUninstalled(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.Packages, name)

	return c.saveToDiskLocked()
}

// MarkUpgraded updates the cache after a package upgrade.
func (c *UnifiedCache) MarkUpgraded(name, newVersion string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.Packages[name]
	state.InstalledVersion = newVersion
	state.LatestVersion = newVersion
	state.Outdated = false
	state.FetchedAt = time.Now()

	c.Packages[name] = state

	return c.saveToDiskLocked()
}

// --- Private methods ---

func (c *UnifiedCache) needsRefreshLocked() bool {
	return c.LastUpdated.IsZero() || time.Since(c.LastUpdated) > c.TTL
}

func (c *UnifiedCache) fetchInstalledPackages() (map[string]bool, error) {
	out, err := c.runner.Run(context.Background(), brewTimeout, runner.Capture, "brew", "list", "--formula", "-1")
	if err != nil {
		return nil, err
	}

	return utils.TransformToMap(
		splitLines(string(out)),
		func(line string) (string, bool) {
			return line, true
		},
	), nil
}

func (c *UnifiedCache) fetchOutdatedPackages() (*brewOutdatedJSON, error) {
	out, err := c.runner.Run(context.Background(), brewTimeout, runner.Capture, "brew", "outdated", "--json=v2")
	if err != nil {
		return nil, err
	}

	// Find first {
	idx := 0
	for i, b := range out {
		if b == '{' {
			idx = i
			break
		}
	}

	var result brewOutdatedJSON
	if err := json.Unmarshal(out[idx:], &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *UnifiedCache) fetchVersionsForChunk(ctx context.Context, names []string) (map[string]PackageInfo, error) {
	if len(names) == 0 {
		return map[string]PackageInfo{}, nil
	}

	args := append([]string{"info", "--json=v2"}, names...)
	out, err := c.runner.Run(ctx, brewTimeout, runner.Capture, "brew", args...)
	if err != nil {
		return nil, err
	}

	var info struct {
		Formulae []struct {
			Name     string `json:"name"`
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
			Installed []struct {
				Version string `json:"version"`
			} `json:"installed"`
		} `json:"formulae"`
	}

	if err := json.Unmarshal(out, &info); err != nil {
		return nil, err
	}

	result := make(map[string]PackageInfo, len(info.Formulae))
	for _, f := range info.Formulae {
		installed := ""
		if len(f.Installed) > 0 {
			installed = f.Installed[0].Version
		}

		result[f.Name] = PackageInfo{
			Name:             f.Name,
			InstalledVersion: installed,
			LatestVersion:    f.Versions.Stable,
		}
	}

	return result, nil
}

func (c *UnifiedCache) loadFromDisk() error {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Fresh start
		}
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := json.Unmarshal(data, c); err != nil {
		logger.Debug("failed to unmarshal cache (will rebuild): %v", err)
		return nil
	}

	logger.Debug("loaded unified cache from disk (%d packages, age: %s)", len(c.Packages), time.Since(c.LastUpdated).Truncate(time.Second))
	return nil
}

func (c *UnifiedCache) saveToDiskLocked() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.cachePath, data, 0o644)
}

// Helper functions

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}

	result := make([]string, 0)
	var line strings.Builder
	line.Grow(64) // Pre-allocate reasonable line size

	for _, r := range s {
		if r == '\n' {
			trimmed := trimSpace(line.String())
			if trimmed != "" {
				result = append(result, trimmed)
			}
			line.Reset()
		} else {
			line.WriteRune(r)
		}
	}

	// Handle last line if no trailing newline
	if line.Len() > 0 {
		trimmed := trimSpace(line.String())
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func chunkStrings(items []string, size int) [][]string {
	if size <= 0 {
		size = 1
	}
	chunks := make([][]string, 0, (len(items)+size-1)/size)
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}
