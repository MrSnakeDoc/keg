package brew

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner implements runner.CommandRunner for testing
type testRunner struct {
	listOutput     string
	listErr        error
	outdatedOutput string
	outdatedErr    error
	infoOutput     string
	infoErr        error
}

func (m *testRunner) Run(ctx context.Context, timeout time.Duration, mode runner.Mode, cmd string, args ...string) ([]byte, error) {
	if cmd == "brew" && len(args) > 0 {
		switch args[0] {
		case "list":
			return []byte(m.listOutput), m.listErr
		case "outdated":
			return []byte(m.outdatedOutput), m.outdatedErr
		case "info":
			return []byte(m.infoOutput), m.infoErr
		}
	}
	return []byte{}, nil
}

func TestUnifiedCache_IsInstalled(t *testing.T) {
	r := &testRunner{
		listOutput: "wget\ncurl\njq\n",
	}

	cache := &UnifiedCache{
		Packages: map[string]PackageState{
			"wget": {Installed: true, FetchedAt: time.Now()},
			"curl": {Installed: true, FetchedAt: time.Now()},
		},
		LastUpdated: time.Now(),
		TTL:         1 * time.Hour,
		runner:      r,
	}

	assert.True(t, cache.IsInstalled("wget"))
	assert.True(t, cache.IsInstalled("curl"))
	assert.False(t, cache.IsInstalled("nonexistent"))
}

func TestUnifiedCache_GetInstalledSet(t *testing.T) {
	cache := &UnifiedCache{
		Packages: map[string]PackageState{
			"wget":   {Installed: true, FetchedAt: time.Now()},
			"curl":   {Installed: true, FetchedAt: time.Now()},
			"notins": {Installed: false, FetchedAt: time.Now()},
		},
	}

	installed := cache.GetInstalledSet()
	assert.Len(t, installed, 2)
	assert.Contains(t, installed, "wget")
	assert.Contains(t, installed, "curl")
	assert.NotContains(t, installed, "notins")
}

func TestUnifiedCache_GetOutdatedMap(t *testing.T) {
	cache := &UnifiedCache{
		Packages: map[string]PackageState{
			"wget": {
				Installed:        true,
				Outdated:         true,
				InstalledVersion: "1.0.0",
				LatestVersion:    "1.1.0",
				FetchedAt:        time.Now(),
			},
			"curl": {
				Installed: true,
				Outdated:  false,
				FetchedAt: time.Now(),
			},
			"jq": {
				Installed:        false,
				Outdated:         true,
				InstalledVersion: "1.5.0",
				LatestVersion:    "1.6.0",
				FetchedAt:        time.Now(),
			},
		},
	}

	outdated := cache.GetOutdatedMap()
	assert.Len(t, outdated, 1, "only installed+outdated packages should be returned")
	assert.Contains(t, outdated, "wget")
	assert.Equal(t, "1.0.0", outdated["wget"].InstalledVersion)
	assert.Equal(t, "1.1.0", outdated["wget"].LatestVersion)
}

func TestUnifiedCache_MarkInstalled(t *testing.T) {
	cache := &UnifiedCache{
		Packages:  make(map[string]PackageState),
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
	}

	err := cache.MarkInstalled("wget", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assert.True(t, cache.Packages["wget"].Installed)
	assert.Equal(t, "1.0.0", cache.Packages["wget"].InstalledVersion)
	assert.Equal(t, "1.0.0", cache.Packages["wget"].LatestVersion)
	assert.False(t, cache.Packages["wget"].Outdated)
}

func TestUnifiedCache_MarkUpgraded(t *testing.T) {
	cache := &UnifiedCache{
		Packages: map[string]PackageState{
			"wget": {
				Installed:        true,
				Outdated:         true,
				InstalledVersion: "1.0.0",
				LatestVersion:    "1.1.0",
			},
		},
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
	}

	err := cache.MarkUpgraded("wget", "1.1.0")
	require.NoError(t, err)

	assert.True(t, cache.Packages["wget"].Installed)
	assert.Equal(t, "1.1.0", cache.Packages["wget"].InstalledVersion)
	assert.Equal(t, "1.1.0", cache.Packages["wget"].LatestVersion)
	assert.False(t, cache.Packages["wget"].Outdated)
}

func TestUnifiedCache_MarkUpgraded_SetsInstalledFlag(t *testing.T) {
	cache := &UnifiedCache{
		Packages:  make(map[string]PackageState),
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
	}

	// Upgrade a package that doesn't exist in cache yet
	err := cache.MarkUpgraded("newpkg", "2.0.0")
	require.NoError(t, err)

	// Should be marked as installed
	assert.True(t, cache.Packages["newpkg"].Installed, "upgraded package should be marked as installed")
	assert.Equal(t, "2.0.0", cache.Packages["newpkg"].InstalledVersion)
}

func TestUnifiedCache_MarkUninstalled(t *testing.T) {
	cache := &UnifiedCache{
		Packages: map[string]PackageState{
			"wget": {
				Installed:        true,
				InstalledVersion: "1.0.0",
			},
		},
		cachePath: filepath.Join(t.TempDir(), "cache.json"),
	}

	err := cache.MarkUninstalled("wget")
	require.NoError(t, err)

	assert.False(t, cache.Packages["wget"].Installed)
	assert.Empty(t, cache.Packages["wget"].InstalledVersion)
	assert.False(t, cache.Packages["wget"].Outdated)
}

func TestUnifiedCache_RefreshVersions_EmptyList(t *testing.T) {
	cache := &UnifiedCache{
		Packages: make(map[string]PackageState),
		runner:   &testRunner{},
	}

	err := cache.RefreshVersions(context.Background(), []string{})
	require.NoError(t, err)
}

func TestSplitLines_HandlesVariousInputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single line",
			input:    "wget",
			expected: []string{"wget"},
		},
		{
			name:     "multiple lines",
			input:    "wget\ncurl\njq",
			expected: []string{"wget", "curl", "jq"},
		},
		{
			name:     "lines with spaces",
			input:    "wget\n  curl  \njq",
			expected: []string{"wget", "curl", "jq"},
		},
		{
			name:     "empty lines",
			input:    "wget\n\ncurl\n\n",
			expected: []string{"wget", "curl"},
		},
		{
			name:     "windows line endings",
			input:    "wget\r\ncurl\r\njq",
			expected: []string{"wget", "curl", "jq"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
