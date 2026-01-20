package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/brew"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

/* -----------------------------
   Test harness + helpers
------------------------------ */

func withIsolatedState(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("XDG_STATE_HOME", tmp)

	// Force cache reset to ensure clean state for each test
	brew.ResetCache()
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
		Data      brewOutdatedJSON `json:"data"`
		Timestamp string           `json:"timestamp"`
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

/* -----------------------------
   Basics: FindPackage / GetName
------------------------------ */

func TestFindPackage_ByCommandAndBinary(t *testing.T) {
	cfg := &models.Config{
		Packages: []models.Package{
			{Command: "foo"},
			{Command: "bar", Binary: "bbar"},
		},
	}
	b := NewBase(cfg, runner.NewMockRunner())

	if p, ok := b.FindPackage("foo"); !ok || p.Command != "foo" {
		t.Fatalf("expected to find foo by command")
	}
	if p, ok := b.FindPackage("bbar"); !ok || p.Command != "bar" {
		t.Fatalf("expected to find bar by binary alias")
	}
	if _, ok := b.FindPackage("nope"); ok {
		t.Fatalf("did not expect to find 'nope'")
	}
}

func TestGetPackageName(t *testing.T) {
	b := NewBase(&models.Config{}, runner.NewMockRunner())
	if got := b.GetPackageName(&models.Package{Command: "foo"}); got != "foo" {
		t.Fatalf("want foo, got %s", got)
	}
	if got := b.GetPackageName(&models.Package{Command: "bar", Binary: "bbar"}); got != "bbar" {
		t.Fatalf("want bbar, got %s", got)
	}
}

/* -----------------------------
   IsPackageInstalled: caching
------------------------------ */

func TestIsPackageInstalled_CachesBrewList(t *testing.T) {
	withIsolatedState(t)

	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo")

	b := NewBase(&models.Config{}, mr)

	// first call → triggers cache refresh
	if !b.IsPackageInstalled("foo") {
		t.Fatalf("expected foo installed")
	}
	// second call → should use cached result (within TTL)
	if !b.IsPackageInstalled("foo") {
		t.Fatalf("expected foo installed on second call")
	}

	// Count brew list calls (should be 1 from initial cache load)
	count := 0
	for _, c := range mr.Commands {
		if c.Name == "brew" && len(c.Args) > 0 && c.Args[0] == "list" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected brew list once, got %d", count)
	}
}

/* -----------------------------
   resolvePackageScoped
------------------------------ */

func TestResolvePackageScoped(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "adhoc") // simulate local install

	cfg := &models.Config{
		Packages: []models.Package{{Command: "foo"}},
	}
	b := NewBase(cfg, mr)

	// present in config
	if p, err := b.resolvePackageScoped("foo", false); err != nil || p.Command != "foo" {
		t.Fatalf("want foo from config, got err=%v p=%+v", err, p)
	}

	// ad-hoc allowed
	if p, err := b.resolvePackageScoped("adhoc", true); err != nil || p.Command != "adhoc" {
		t.Fatalf("want adhoc synthesized, got err=%v p=%+v", err, p)
	}

	// not in config + not installed + no ad-hoc
	if _, err := b.resolvePackageScoped("ghost", false); err == nil || !strings.Contains(err.Error(), "package not found") {
		t.Fatalf("expected ErrPkgNotFound, got %v", err)
	}
}

/* -----------------------------
   HandlePackages: explicit args
------------------------------ */

func TestHandlePackages_WithArgs_Install(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	cfg := &models.Config{Packages: []models.Package{{Command: "foo"}, {Command: "bar"}}}
	b := NewBase(cfg, mr)

	opts := PackageHandlerOptions{
		Action: PackageAction{ActionVerb: "install"},
		Packages: []string{
			"foo",
		},
		FilterFunc:   func(*models.Package) bool { return true },
		ValidateFunc: func(string) bool { return true },
	}

	if err := b.HandlePackages(opts); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !mr.VerifyCommand("brew", "install", "foo") {
		t.Fatalf("expected brew install foo, got %+v", mr.Commands)
	}
}

/* -----------------------------
   HandlePackages: config loop, skip optional
------------------------------ */

func TestHandlePackages_ConfigLoop_SkipOptional(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	cfg := &models.Config{
		Packages: []models.Package{
			{Command: "a"},
			{Command: "b", Optional: true},
			{Command: "c"},
		},
	}
	b := NewBase(cfg, mr)

	opts := DefaultPackageHandlerOptions(PackageAction{ActionVerb: "install"})
	// Default FilterFunc skips optional

	if err := b.HandlePackages(opts); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	got := strings.Join(flattenCmds(mr), ";")
	if !strings.Contains(got, "brew install a") || !strings.Contains(got, "brew install c") {
		t.Fatalf("missing expected installs a/c, got: %s", got)
	}
	if strings.Contains(got, "brew install b") {
		t.Fatalf("should have skipped optional b")
	}
}

func flattenCmds(m *runner.MockRunner) []string {
	out := make([]string, 0, len(m.Commands))
	for _, c := range m.Commands {
		out = append(out, c.Name+" "+strings.Join(c.Args, " "))
	}
	return out
}

/* -----------------------------
   HandlePackages: ValidateFunc
------------------------------ */

func TestHandlePackages_ValidateRejects(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	cfg := &models.Config{Packages: []models.Package{{Command: "foo"}}}
	b := NewBase(cfg, mr)

	opts := PackageHandlerOptions{
		Action:       PackageAction{ActionVerb: "install"},
		FilterFunc:   func(*models.Package) bool { return true },
		ValidateFunc: func(string) bool { return false },
	}

	if err := b.HandlePackages(opts); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	// Should have cache refresh calls (brew list, brew outdated) but no install
	hasInstall := false
	for _, c := range mr.Commands {
		if c.Name == "brew" && len(c.Args) > 0 && c.Args[0] == "install" {
			hasInstall = true
		}
	}
	if hasInstall {
		t.Fatalf("expected no install calls (ValidateFunc rejected), got %+v", mr.Commands)
	}
}

/* -----------------------------
   HandlePackages: ad-hoc allowed (install)
------------------------------ */

func TestHandlePackages_AdHocAllowed(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "dep")

	cfg := &models.Config{Packages: []models.Package{{Command: "foo"}}}
	b := NewBase(cfg, mr)

	opts := PackageHandlerOptions{
		Action:       PackageAction{ActionVerb: "install"},
		Packages:     []string{"dep"},
		AllowAdHoc:   true,
		ValidateFunc: func(string) bool { return true },
	}

	if err := b.HandlePackages(opts); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !mr.VerifyCommand("brew", "install", "dep") {
		t.Fatalf("expected brew install dep, got %+v", mr.Commands)
	}
}

/* -----------------------------
   Upgrade flow: only when outdated
------------------------------ */

func TestHandlePackages_Upgrade_OnlyWhenOutdated(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo")

	// Mock brew info to show versions
	mr.MockBrewInfoV2Formula("foo", "1.0.0", "1.1.0")

	// Mock outdated JSON showing foo is outdated
	mr.AddResponse("brew|outdated|--json=v2", []byte(`{"formulae":[{"name":"foo","installed_versions":["1.0.0"],"current_version":"1.1.0"}],"casks":[]}`), nil)

	cfg := &models.Config{Packages: []models.Package{{Command: "foo"}}}
	b := NewBase(cfg, mr)

	opts := PackageHandlerOptions{
		Action:       PackageAction{ActionVerb: "upgrade"},
		Packages:     []string{"foo"},
		ValidateFunc: func(string) bool { return true },
	}

	if err := b.HandlePackages(opts); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !mr.VerifyCommand("brew", "upgrade", "foo") {
		t.Fatalf("expected brew upgrade foo, got %+v", mr.Commands)
	}
}

func TestHandlePackages_Upgrade_SkipsWhenNotOutdated(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo")

	// No outdated entries
	writeOutdatedCache(t, map[string][2]string{})

	cfg := &models.Config{Packages: []models.Package{{Command: "foo"}}}
	b := NewBase(cfg, mr)

	opts := PackageHandlerOptions{
		Action:       PackageAction{ActionVerb: "upgrade"},
		Packages:     []string{"foo"},
		ValidateFunc: func(string) bool { return true },
	}

	if err := b.HandlePackages(opts); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// Should not see brew upgrade foo
	for _, c := range mr.Commands {
		if c.Name == "brew" && len(c.Args) >= 2 && c.Args[0] == "upgrade" && c.Args[1] == "foo" {
			t.Fatalf("did not expect upgrade call, got %+v", mr.Commands)
		}
	}
}

/* -----------------------------
   SkipMessage short-circuit
------------------------------ */

func TestHandleSelectedPackage_SkipMessageWhenInstalled(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()
	primeInstalled(mr, "foo")

	cfg := &models.Config{Packages: []models.Package{{Command: "foo"}}}
	b := NewBase(cfg, mr)

	action := PackageAction{
		ActionVerb:  "install",
		SkipMessage: "%s already installed",
	}
	if err := b.handleSelectedPackage(action, "foo", func(string) bool { return true }, false); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	// No brew install should be called
	for _, c := range mr.Commands {
		if c.Name == "brew" && len(c.Args) >= 2 && c.Args[0] == "install" && c.Args[1] == "foo" {
			t.Fatalf("did not expect install, got %+v", mr.Commands)
		}
	}
}

/* -----------------------------
   touchVersionCache: Touch & Remove
------------------------------ */

func TestTouchVersionCache_Remove(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()

	// Mock empty list (package not installed)
	mr.AddResponse("brew|list|--formula|-1", []byte(""), nil)

	b := NewBase(&models.Config{}, mr)

	// This should handle removal gracefully (package not installed)
	b.touchVersionCache("gone")

	// Verify cache reflects the removal
	if b.cache != nil && b.cache.IsInstalled("gone") {
		t.Fatalf("expected 'gone' to not be marked as installed in unified cache")
	}
}

func TestTouchVersionCache_Touch(t *testing.T) {
	withIsolatedState(t)
	mr := runner.NewMockRunner()

	// Mock foo being installed
	mr.AddResponse("brew|list|--formula|-1", []byte("foo\n"), nil)
	mr.MockBrewInfoV2Formula("foo", "1.2.3", "1.2.3")

	b := NewBase(&models.Config{}, mr)
	b.touchVersionCache("foo")

	// Verify the unified cache reflects the installation
	if b.cache == nil || !b.cache.IsInstalled("foo") {
		t.Fatalf("expected foo to be marked as installed in unified cache")
	}
}
