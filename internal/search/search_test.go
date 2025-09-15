package search_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/index"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/search"
	"github.com/MrSnakeDoc/keg/internal/store"
)

/*
---------------------------------
  Test harness
---------------------------------
*/

func TestMain(m *testing.M) {
	// Silence logs but keep functionality intact
	logger.UseTestMode()
	os.Exit(m.Run())
}

type structDef struct {
	name           string
	items          []index.ItemLight
	args           []string
	exact          bool
	noDesc         bool
	regex          bool
	fzf            bool
	jsonOut        bool
	limit          int
	refresh        bool
	expectError    string
	expectJSONLen  int
	expectContains []string
}

var searchTestCases = []structDef{
	{
		name:          "JSON no query returns all",
		items:         []index.ItemLight{{Name: "foo"}, {Name: "bar"}},
		jsonOut:       true,
		expectJSONLen: 2,
	},
	{
		name:           "Query matches name",
		items:          []index.ItemLight{{Name: "foo"}, {Name: "baz"}},
		args:           []string{"foo"},
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"foo"},
	},
	{
		name:           "Exact match only foo",
		items:          []index.ItemLight{{Name: "foo"}, {Name: "foobar"}},
		args:           []string{"foo"},
		exact:          true,
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"foo"},
	},
	{
		name:           "Regex match ^a",
		items:          []index.ItemLight{{Name: "abc"}, {Name: "xyz"}},
		args:           []string{"^a"},
		regex:          true,
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"abc"},
	},
	{
		name:          "NoDesc excludes description matches",
		items:         []index.ItemLight{{Name: "zap", Desc: "Foo package"}},
		args:          []string{"Foo"},
		noDesc:        true,
		jsonOut:       true,
		expectJSONLen: 0,
	},
	{
		name:          "Limit to 2 results",
		items:         []index.ItemLight{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		limit:         2,
		jsonOut:       true,
		expectJSONLen: 2,
	},
	{
		name:           "FZF TSV output",
		items:          []index.ItemLight{{Name: "alpha", Aliases: []string{"a1"}, Desc: "Alpha"}},
		fzf:            true,
		expectContains: []string{"alpha\ta1\tAlpha"},
	},
	{
		name:  "Error on --json + --fzf",
		items: []index.ItemLight{{Name: "foo"}},
		fzf:   true, jsonOut: true,
		expectError: "cannot use --json and --fzf together",
	},
	{
		name:  "Invalid regex pattern",
		items: []index.ItemLight{{Name: "x"}},
		args:  []string{"["}, regex: true,
		expectError: "invalid regex",
	},
	{
		name:    "JSON marshal error",
		args:    []string{"pkg"},
		jsonOut: true,
	},
	{
		name:           "fzf output should contain result",
		items:          []index.ItemLight{{Name: "pkg", Desc: "Dummy"}},
		fzf:            true,
		expectContains: []string{"pkg\t\tDummy"},
	},
	{
		name:          "No description match with noDesc=true",
		args:          []string{"desc"},
		noDesc:        true,
		expectJSONLen: 0,
	},

	{
		name:           "Exact match via alias",
		items:          []index.ItemLight{{Name: "foo", Aliases: []string{"bar"}}},
		args:           []string{"bar"},
		exact:          true,
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"foo"},
	},
	{
		name:           "Exact match via oldname",
		items:          []index.ItemLight{{Name: "foo", OldNames: []string{"baz"}}},
		args:           []string{"baz"},
		exact:          true,
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"foo"},
	},
	{
		name:           "Desc included by default",
		items:          []index.ItemLight{{Name: "abc", Desc: "lorem ipsum"}},
		args:           []string{"lorem"},
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"abc"},
	},
	{
		name:          "NoDesc excludes description matches",
		items:         []index.ItemLight{{Name: "abc", Desc: "lorem ipsum"}},
		args:          []string{"lorem"},
		noDesc:        true,
		jsonOut:       true,
		expectJSONLen: 0,
	},
	{
		name:           "Limit=0 returns all items",
		items:          []index.ItemLight{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		limit:          0,
		jsonOut:        true,
		expectJSONLen:  3,
		expectContains: []string{"one", "two", "three"},
	},
	{
		name:           "Limit==len returns all items",
		items:          []index.ItemLight{{Name: "one"}, {Name: "two"}, {Name: "three"}},
		limit:          3,
		jsonOut:        true,
		expectJSONLen:  3,
		expectContains: []string{"one", "two", "three"},
	},
	{
		name:           "FZF TSV without aliases",
		items:          []index.ItemLight{{Name: "solo", Desc: "Only"}},
		fzf:            true,
		expectContains: []string{"solo\t\tOnly"},
	},
	{
		name:           "FZF TSV with multiple aliases",
		items:          []index.ItemLight{{Name: "alpha", Aliases: []string{"a1", "a2"}, Desc: "Alpha"}},
		fzf:            true,
		expectContains: []string{"alpha\ta1,a2\tAlpha"},
	},
	{
		name:           "Regex safe pattern matches one",
		items:          []index.ItemLight{{Name: "foo"}, {Name: "bar"}},
		args:           []string{"^fo.*"},
		regex:          true,
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"foo"},
	},
	{
		name:           "Case-insensitive substring match",
		items:          []index.ItemLight{{Name: "foo"}, {Name: "bar"}},
		args:           []string{"FOO"},
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"foo"},
	},
	{
		name:           "Substring match in aliases",
		items:          []index.ItemLight{{Name: "tool", Aliases: []string{"fancytool"}}},
		args:           []string{"fancy"},
		jsonOut:        true,
		expectJSONLen:  1,
		expectContains: []string{"tool"},
	},
	{
		name:           "No query returns all (sanity check)",
		items:          []index.ItemLight{{Name: "x"}, {Name: "y"}},
		jsonOut:        true,
		expectJSONLen:  2,
		expectContains: []string{"x", "y"},
	},
}

// runSearch executes Searcher.Execute with the test case parameters
// and captures stdout.
func runSearch(t *testing.T, tt structDef, s *search.Searcher, cfg *models.Config) (string, error) {
	t.Helper()
	out := captureOutput(func() {
		_ = s.Execute(tt.args, nil, cfg,
			tt.exact, tt.noDesc, tt.regex, tt.fzf, tt.jsonOut,
			tt.limit, tt.refresh, true)
	})
	// We need to re-run to capture the error (Execute only inside closure).
	err := s.Execute(tt.args, nil, cfg,
		tt.exact, tt.noDesc, tt.regex, tt.fzf, tt.jsonOut,
		tt.limit, tt.refresh, true)
	return out, err
}

// checkSearchResult validates output against expectations in tt.
func checkSearchResult(t *testing.T, tt structDef, out string, err error) {
	t.Helper()

	if tt.expectError != "" {
		if err == nil || !strings.Contains(err.Error(), tt.expectError) {
			t.Fatalf("expected error %q, got %v", tt.expectError, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if tt.jsonOut {
		var got []index.ItemLight
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, out)
		}
		if len(got) != tt.expectJSONLen {
			t.Fatalf("expected %d items, got %d: %+v",
				tt.expectJSONLen, len(got), got)
		}
		for _, substr := range tt.expectContains {
			if !strings.Contains(out, substr) {
				t.Errorf("expected output to contain %q, got %s", substr, out)
			}
		}
	}

	if tt.fzf && !tt.jsonOut {
		for _, substr := range tt.expectContains {
			if !strings.Contains(out, substr) {
				t.Errorf("expected fzf output to contain %q, got %s", substr, out)
			}
		}
	}
}

func TestSearcher_Execute_Table(t *testing.T) {
	for _, tt := range searchTestCases {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestSearcher(t, tt.items)
			cfg := &models.Config{}
			out, err := runSearch(t, tt, s, cfg)
			checkSearchResult(t, tt, out, err)
		})
	}
}

//--------------------------------
//  Test helpers
//--------------------------------

// captureOutput captures stdout of the provided function.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

// writeGzIndex writes an IndexLight gzipped payload + meta using store.FS API.
func writeGzIndex(t *testing.T, fs *store.FS, items []index.ItemLight) {
	t.Helper()

	payload := index.IndexLight{Items: items}

	var raw bytes.Buffer
	gzw := gzip.NewWriter(&raw)
	if err := json.NewEncoder(gzw).Encode(payload); err != nil {
		t.Fatalf("encode gz index: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gz writer: %v", err)
	}

	meta := store.Meta{
		ETag:        "test-etag",
		GeneratedAt: time.Now().UTC(),
		SizeBytes:   int64(raw.Len()),
		// Count/Sha256 optional; leave zero-value
	}

	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(raw.Bytes()), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}
}

// newTestSearcher creates a temp data dir, FS store, writes a fake gz index,
// and returns a Searcher wired to that store (no HTTP client).
func newTestSearcher(t *testing.T, items []index.ItemLight) *search.Searcher {
	t.Helper()
	tmp := t.TempDir()

	// data dir is exactly what store.NewFS expects (no globalconfig in tests)
	fs, err := store.NewFS(tmp)
	if err != nil {
		t.Fatalf("store.NewFS: %v", err)
	}
	writeGzIndex(t, fs, items)
	return search.New(fs, nil)
}
