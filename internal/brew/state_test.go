package brew

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

func minimalOutdatedJSON(t *testing.T, m map[string][2]string) []byte {
	t.Helper()
	type F struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	}
	type Root struct {
		Formulae []F   `json:"formulae"`
		Casks    []any `json:"casks"`
	}
	r := Root{Formulae: []F{}, Casks: []any{}}
	for name, pair := range m {
		r.Formulae = append(r.Formulae, F{
			Name: name, InstalledVersions: []string{pair[0]}, CurrentVersion: pair[1],
		})
	}
	b, _ := json.Marshal(r)
	return b
}

func withIsolatedState(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("XDG_STATE_HOME", tmp)
}

func TestFetchState_ParseOK(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()

	// brew list --formula -1
	mr.AddResponse("brew|list|--formula|-1", []byte("foo\nbar\n"), nil)

	// brew outdated --json=v2
	outJSON := minimalOutdatedJSON(t, map[string][2]string{"foo": {"1.0.0", "1.1.0"}})
	prev := mr.ResponseFunc
	mr.ResponseFunc = func(name string, args ...string) ([]byte, error) {
		if name == "brew" && len(args) >= 2 && args[0] == "outdated" {
			return outJSON, nil
		}
		if prev != nil {
			return prev(name, args...)
		}
		return []byte{}, nil
	}

	st, err := FetchState(mr)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if _, ok := st.Installed["foo"]; !ok {
		t.Fatal("foo should be installed")
	}
	if _, ok := st.Installed["bar"]; !ok {
		t.Fatal("bar should be installed")
	}
	if v, ok := st.Outdated["foo"]; !ok || v.LatestVersion != "1.1.0" {
		t.Fatalf("want foo outdated->1.1.0, got: %#v", v)
	}
}

func TestFetchState_BadJSON(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	mr.GetBrewList("foo")
	mr.ResponseFunc = func(name string, args ...string) ([]byte, error) {
		if name == "brew" && len(args) >= 2 && args[0] == "outdated" {
			return []byte(`{ this is: not-json`), nil
		}
		return []byte{}, nil
	}
	if _, err := FetchState(mr); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func readOutdatedCache(t *testing.T) cacheFile {
	t.Helper()
	cachePath := filepath.Join(os.Getenv("HOME"), utils.CacheDir, utils.OutdatedFile)
	b, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}

	var cache cacheFile
	if err := json.Unmarshal(b, &cache); err != nil {
		t.Fatalf("decode cache: %v", err)
	}
	return cache
}

func countOutdatedCalls(mr *runner.MockRunner) int {
	count := 0
	for _, command := range mr.Commands {
		if command.Name == "brew" && len(command.Args) > 0 && command.Args[0] == "outdated" {
			count++
		}
	}
	return count
}

func TestFetchOutdatedPackages_PersistsAndReusesCache(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	mr.AddResponse("brew|list|--formula|-1", []byte("foo\n"), nil)
	outJSON := minimalOutdatedJSON(t, map[string][2]string{"foo": {"1.0.0", "1.1.0"}})
	mr.ResponseFunc = func(name string, args ...string) ([]byte, error) {
		if name == "brew" && len(args) > 0 && args[0] == "outdated" {
			return outJSON, nil
		}
		return []byte{}, nil
	}

	first, err := FetchState(mr)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}
	if got := first.Outdated["foo"].LatestVersion; got != "1.1.0" {
		t.Fatalf("first fetch returned latest version %q, want %q", got, "1.1.0")
	}

	cache := readOutdatedCache(t)
	if cache.Data == nil || len(cache.Data.Formulae) != 1 {
		t.Fatalf("cache data was not persisted: %#v", cache.Data)
	}
	if cache.Timestamp.IsZero() {
		t.Fatal("cache timestamp was not persisted")
	}

	second, err := FetchState(mr)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if got := second.Outdated["foo"].LatestVersion; got != "1.1.0" {
		t.Fatalf("second fetch returned latest version %q, want %q", got, "1.1.0")
	}

	outdatedCalls := countOutdatedCalls(mr)
	if outdatedCalls != 1 {
		t.Fatalf("brew outdated was called %d times, want 1", outdatedCalls)
	}
}
