package brew

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/MrSnakeDoc/keg/internal/runner"
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
	// FetchState now returns empty maps on JSON error instead of failing
	st, err := FetchState(mr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have installed packages but no outdated (due to JSON error)
	if len(st.Installed) == 0 {
		t.Fatal("expected at least installed packages")
	}
}
