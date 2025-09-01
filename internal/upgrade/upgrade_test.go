package upgrade

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

/*
-----------------------------

	TestMain: silence logs everywhere

------------------------------
*/
func TestMain(m *testing.M) {
	logger.UseTestMode()
	os.Exit(m.Run())
}

/*
-----------------------------

   	Helpers: state, cache, stubs

------------------------------
*/

func withIsolatedState(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("XDG_STATE_HOME", tmp)
}

func writeOutdatedCache(t *testing.T, entries map[string][2]string) {
	t.Helper()
	home := os.Getenv("HOME")
	stateDir := filepath.Join(home, ".local", "state", "keg")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	cachePath := filepath.Join(stateDir, "keg_brew_update.json")

	type outdatedFormula struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	}
	type brewOutdatedJSON struct {
		Formulae []outdatedFormula `json:"formulae"`
		Casks    []any             `json:"casks"`
	}
	type cacheFile struct {
		Data      brewOutdatedJSON `json:"Data"`
		Timestamp string           `json:"Timestamp"`
	}

	payload := brewOutdatedJSON{Formulae: make([]outdatedFormula, 0, len(entries)), Casks: []any{}}
	for name, pair := range entries {
		payload.Formulae = append(payload.Formulae, outdatedFormula{
			Name:              name,
			InstalledVersions: []string{pair[0]},
			CurrentVersion:    pair[1],
		})
	}
	wrapper := cacheFile{
		Data:      payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	b, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	if err := os.WriteFile(cachePath, b, 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}
}

func primeInstalled(mr *runner.MockRunner, pkgs ...string) {
	listOut := strings.Join(pkgs, "\n") + "\n"
	mr.GetBrewList(pkgs...)
	prev := mr.ResponseFunc
	mr.ResponseFunc = func(name string, args ...string) ([]byte, error) {
		if name == "brew" && len(args) > 0 && args[0] == "list" {
			return []byte(listOut), nil
		}
		if prev != nil {
			return prev(name, args...)
		}
		return []byte{}, nil
	}
}

func flattenCmds(m *runner.MockRunner) []string {
	out := make([]string, 0, len(m.Commands))
	for _, c := range m.Commands {
		out = append(out, c.Name+" "+strings.Join(c.Args, " "))
	}
	return out
}

func sawUpgrade(m *runner.MockRunner, pkg string) bool {
	for _, c := range m.Commands {
		if c.Name == "brew" && len(c.Args) >= 2 && c.Args[0] == "upgrade" && c.Args[1] == pkg {
			return true
		}
	}
	return false
}

/* -----------------------------
   Tests: CheckUpgrades
------------------------------ */

func TestCheckUpgrades_ManifestOnly(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo")
	writeOutdatedCache(t, map[string][2]string{})

	cfg := models.Config{Packages: []models.Package{{Command: "foo"}}}
	up := New(&cfg, mr)

	if err := up.CheckUpgrades(nil, false); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestCheckUpgrades_WithAll_IncludesDeps(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo", "dep")
	writeOutdatedCache(t, map[string][2]string{
		"dep": {"0.9.0", "1.0.0"},
	})

	cfg := models.Config{Packages: []models.Package{{Command: "foo"}}}
	up := New(&cfg, mr)

	if err := up.CheckUpgrades(nil, true); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestCheckUpgrades_WithArgs_SingleAndMultiple(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo", "bar")
	writeOutdatedCache(t, map[string][2]string{
		"foo": {"1.0.0", "1.1.0"},
	})

	cfg := models.Config{Packages: []models.Package{
		{Command: "foo"},
		{Command: "bar"},
	}}
	up := New(&cfg, mr)

	if err := up.CheckUpgrades([]string{"foo"}, false); err != nil {
		t.Fatalf("unexpected err(single): %v", err)
	}
	if err := up.CheckUpgrades([]string{"foo", "bar"}, false); err != nil {
		t.Fatalf("unexpected err(multi): %v", err)
	}
}

/* -----------------------------
   Tests: Execute (upgrade)
------------------------------ */

func TestExecute_WithArgs_UpgradesSingle(t *testing.T) {
	withIsolatedState(t)
	cfg := models.Config{Packages: []models.Package{{Command: "foo"}}}
	mr := runner.NewMockRunner()

	primeInstalled(mr, "foo")
	writeOutdatedCache(t, map[string][2]string{
		"foo": {"1.0.0", "1.1.0"},
	})

	up := New(&cfg, mr)

	if err := up.Execute([]string{"foo"}, false, false); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !sawUpgrade(mr, "foo") {
		t.Fatalf("expected 'brew upgrade foo', got: %#v", mr.Commands)
	}
}

func TestExecute_WithMultipleArgs_UpgradesAll(t *testing.T) {
	withIsolatedState(t)
	cfg := models.Config{Packages: []models.Package{
		{Command: "foo"},
		{Command: "bar"},
		{Command: "baz"},
	}}
	mr := runner.NewMockRunner()

	primeInstalled(mr, "foo", "bar", "baz")
	writeOutdatedCache(t, map[string][2]string{
		"foo": {"1.0.0", "1.1.0"},
		"bar": {"1.0.0", "1.1.0"},
		"baz": {"1.0.0", "1.1.0"},
	})

	up := New(&cfg, mr)

	if err := up.Execute([]string{"foo", "bar", "baz"}, false, false); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	got := strings.Join(flattenCmds(mr), ";")
	for _, want := range []string{
		"brew upgrade foo",
		"brew upgrade bar",
		"brew upgrade baz",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in calls: %s", want, got)
		}
	}
}

func TestExecute_All_ManifestOnly(t *testing.T) {
	withIsolatedState(t)
	cfg := models.Config{Packages: []models.Package{
		{Command: "foo"},
		{Command: "bar"},
	}}
	mr := runner.NewMockRunner()

	primeInstalled(mr, "foo", "bar")
	writeOutdatedCache(t, map[string][2]string{
		"foo": {"1.0.0", "1.1.0"},
		"bar": {"2.0.0", "2.1.0"},
	})

	up := New(&cfg, mr)

	if err := up.Execute(nil, false, true); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !sawUpgrade(mr, "foo") || !sawUpgrade(mr, "bar") {
		t.Fatalf("expected upgrades for foo+bar, got: %#v", mr.Commands)
	}
}

func TestExecute_All_IncludesDeps(t *testing.T) {
	withIsolatedState(t)
	cfg := models.Config{Packages: []models.Package{{Command: "foo"}}}
	mr := runner.NewMockRunner()

	// installed: foo (manifest) + dep (ad-hoc)
	primeInstalled(mr, "foo", "dep")
	writeOutdatedCache(t, map[string][2]string{
		"foo": {"1.0.0", "1.1.0"},
		"dep": {"0.9.0", "1.0.0"},
	})

	up := New(&cfg, mr)

	if err := up.Execute(nil, false, true); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	got := strings.Join(flattenCmds(mr), ";")
	for _, want := range []string{"brew upgrade foo", "brew upgrade dep"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in calls: %s", want, got)
		}
	}
}

func TestExecute_OptionalNotInstalled_IsSkipped(t *testing.T) {
	withIsolatedState(t)
	cfg := models.Config{Packages: []models.Package{
		{Command: "opt", Optional: true},
	}}
	mr := runner.NewMockRunner()

	primeInstalled(mr /* rien */)
	writeOutdatedCache(t, map[string][2]string{})

	up := New(&cfg, mr)

	if err := up.Execute(nil, false, false); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if sawUpgrade(mr, "opt") || strings.Contains(strings.Join(flattenCmds(mr), ";"), "brew upgrade ") {
		t.Fatalf("expected no 'brew upgrade' calls, got: %#v", mr.Commands)
	}
}
