package upgrade

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/MrSnakeDoc/keg/internal/brew"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/runner"
)

func setInstalled(up *Upgrader, names ...string) {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	v := reflect.ValueOf(up.Base).Elem().FieldByName("installedPkgs")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(m))
}

func TestSortPackages(t *testing.T) {
	cfg := models.Config{
		Packages: []models.Package{
			{Command: "foo"},
			{Command: "bar"},
		},
	}
	up := New(&cfg, runner.NewMockRunner())

	state := &brew.BrewState{
		Installed: map[string]string{
			"foo": "", "bar": "", "dep": "",
		},
	}
	got := up.sortPackages(state)

	expCfg := []string{"foo", "bar"}
	expDeps := []string{"dep"}

	if !reflect.DeepEqual(got.configured, expCfg) {
		t.Errorf("configured: want %v, got %v", expCfg, got.configured)
	}
	if !reflect.DeepEqual(got.deps, expDeps) {
		t.Errorf("deps: want %v, got %v", expDeps, got.deps)
	}
}

func TestCheckSinglePackage(t *testing.T) {
	up := New(&models.Config{}, runner.NewMockRunner())
	setInstalled(up, "foo")

	vm := map[string]brew.PackageInfo{
		"foo": {Name: "foo", InstalledVersion: "1.0.0"},
	}
	if err := up.checkSinglePackage("foo", vm); err != nil {
		t.Fatalf("outdated: unexpected err %v", err)
	}

	if err := up.checkSinglePackage("foo", map[string]brew.PackageInfo{}); err != nil {
		t.Fatalf("uptodate: unexpected err %v", err)
	}

	if err := up.checkSinglePackage("bar", map[string]brew.PackageInfo{}); err == nil {
		t.Fatalf("expected error for non-installed pkg")
	}
}
