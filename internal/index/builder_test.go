package index

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

/*
   Helpers
*/

// --- generic, table-driven test helpers ---

type checkFn func(ItemLight) error

func runChecks(t *testing.T, it ItemLight, checks []checkFn) {
	t.Helper()
	for i, c := range checks {
		if err := c(it); err != nil {
			t.Errorf("check #%d: %v", i+1, err)
		}
	}
}

func eqStr(label, want string, get func(ItemLight) string) checkFn {
	return func(it ItemLight) error {
		got := get(it)
		if got != want {
			return fmt.Errorf("%s = %q, want %q", label, got, want)
		}
		return nil
	}
}

func eqBool(label string, want bool, get func(ItemLight) bool) checkFn {
	return func(it ItemLight) error {
		got := get(it)
		if got != want {
			return fmt.Errorf("%s = %v, want %v", label, got, want)
		}
		return nil
	}
}

func eqInt(label string, want int, get func(ItemLight) int) checkFn {
	return func(it ItemLight) error {
		got := get(it)
		if got != want {
			return fmt.Errorf("%s = %d, want %d", label, got, want)
		}
		return nil
	}
}

func lenStrs(label string, want int, get func(ItemLight) []string) checkFn {
	return func(it ItemLight) error {
		got := len(get(it))
		if got != want {
			return fmt.Errorf("len(%s) = %d, want %d", label, got, want)
		}
		return nil
	}
}

// Custom compound checks when several fields must be validated together.
func checkDeprecation(wantFlag bool, date, reason, repl string) checkFn {
	return func(it ItemLight) error {
		if it.Deprecated != wantFlag ||
			it.DeprecationDate != date ||
			it.DeprecationReason != reason ||
			it.Replacement != repl {
			return fmt.Errorf("deprecation mismatch: %+v", it)
		}
		return nil
	}
}

func checkDisabled(wantFlag bool, date, reason string) checkFn {
	return func(it ItemLight) error {
		if it.Disabled != wantFlag || it.DisableDate != date || it.DisableReason != reason {
			return fmt.Errorf("disabled mismatch: %+v", it)
		}
		return nil
	}
}

func mustGzip(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(b); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func upstreamArrayJSON(items ...string) []byte {
	return []byte("[" + strings.Join(items, ",") + "]")
}

func decodeResultGzip(t *testing.T, gz []byte) (raw string, idx IndexLight) {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		t.Fatalf("gunzip result: %v", err)
	}
	defer func() { _ = gr.Close() }()

	all, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("read gunzipped result: %v", err)
	}
	raw = string(all)
	if err := json.Unmarshal(all, &idx); err != nil {
		t.Fatalf("unmarshal IndexLight: %v\nraw=%s", err, raw)
	}
	return raw, idx
}

// ReadCloser that returns a chosen error on Close().
type rcWithCloseErr struct {
	io.Reader
	closeErr error
}

func (r rcWithCloseErr) Close() error { return r.closeErr }

/*
   Tests
*/

func assertFooMapping(t *testing.T, it ItemLight) {
	t.Helper()
	runChecks(t, it, []checkFn{
		eqStr("name", "foo", func(it ItemLight) string { return it.Name }),
		eqStr("full_name", "homebrew/core/foo", func(it ItemLight) string { return it.FullName }),
		eqStr("version", "1.2.3", func(it ItemLight) string { return it.Version }),
		eqStr("desc", "<b>bold</b>", func(it ItemLight) string { return it.Desc }),
		checkDeprecation(true, "2024-01-01", "renamed", "bar"),
		checkDisabled(true, "2024-02-01", "broken"),
		eqBool("keg_only", true, func(it ItemLight) bool { return it.KegOnly }),
		eqBool("has_bottle", true, func(it ItemLight) bool { return it.HasBottle }),
		eqInt("dep_count", 3, func(it ItemLight) int { return it.DepCount }),
		eqBool("outdated", false, func(it ItemLight) bool { return it.Outdated }),
		eqBool("pinned", true, func(it ItemLight) bool { return it.Pinned }),
		lenStrs("aliases", 2, func(it ItemLight) []string { return it.Aliases }),
		lenStrs("oldnames", 1, func(it ItemLight) []string { return it.OldNames }),
	})
}

func assertXyzMapping(t *testing.T, it ItemLight) {
	t.Helper()
	runChecks(t, it, []checkFn{
		eqStr("name", "xyz", func(it ItemLight) string { return it.Name }),
		eqStr("version", "0.9.0", func(it ItemLight) string { return it.Version }),
		eqStr("replacement", "xyz-cask", func(it ItemLight) string { return it.Replacement }),
		eqBool("has_bottle", false, func(it ItemLight) bool { return it.HasBottle }),
		eqInt("dep_count", 0, func(it ItemLight) int { return it.DepCount }),
		eqBool("outdated", true, func(it ItemLight) bool { return it.Outdated }),
		eqBool("pinned", false, func(it ItemLight) bool { return it.Pinned }),
	})
}

func buildTwoItemInput() []byte {
	item1 := `{
		"name":"foo","full_name":"homebrew/core/foo",
		"aliases":["f1","f2"],"oldnames":["foo-old"],
		"desc":"<b>bold</b>","tap":"homebrew/core",
		"homepage":"https://example.com","license":"MIT",
		"versions":{"stable":"1.2.3"},
		"deprecated":true,"deprecation_date":"2024-01-01",
		"deprecation_reason":"renamed",
		"deprecation_replacement_formula":"bar",
		"disabled":true,"disable_date":"2024-02-01","disable_reason":"broken",
		"keg_only":true,
		"bottle":{"stable":{"files":{"darwin":{"cellar":"any","url":"http://u","sha256":"deadbeef"}}}},
		"dependencies":["a","b","c"],
		"outdated":false,"pinned":true
	}`
	item2 := `{
		"name":"xyz","full_name":"homebrew/core/xyz",
		"aliases":[],"oldnames":[],
		"desc":"plain","tap":"homebrew/core",
		"homepage":"https://xyz.example.com","license":"Apache-2.0",
		"versions":{"stable":"0.9.0"},
		"deprecated":true,"deprecation_date":"2023-10-10",
		"deprecation_reason":"superseded",
		"deprecation_replacement_cask":"xyz-cask",
		"disabled":false,"disable_date":"","disable_reason":"",
		"keg_only":false,
		"bottle":{"stable":{"files":{}}},
		"dependencies":[],
		"outdated":true,"pinned":false
	}`
	return upstreamArrayJSON(item1, item2)
}

func TestBuildLightIndex_Uncompressed_Mapping(t *testing.T) {
	// Build once
	input := buildTwoItemInput()
	src := io.NopCloser(bytes.NewReader(input))

	res, err := BuildLightIndex(context.Background(), src)
	if err != nil {
		t.Fatalf("BuildLightIndex: %v", err)
	}

	// Assert meta
	assertMeta(t, res, 2)

	// Decode & assert records
	raw, idx := decodeResultGzip(t, res.Gzip)
	if len(idx.Items) != 2 {
		t.Fatalf("decoded items len = %d, want 2; raw=%s", len(idx.Items), raw)
	}

	t.Run("item1-foo", func(t *testing.T) { assertFooMapping(t, idx.Items[0]) })
	t.Run("item2-xyz", func(t *testing.T) { assertXyzMapping(t, idx.Items[1]) })
	t.Run("no-html-escape", func(t *testing.T) { assertNoHTMLEscape(t, raw) })
}

func assertMeta(t *testing.T, res Result, wantCount int) {
	t.Helper()
	if res.Count != wantCount {
		t.Fatalf("res.Count = %d, want %d", res.Count, wantCount)
	}
	if res.SizeBytes != int64(len(res.Gzip)) {
		t.Fatalf("res.SizeBytes mismatch: got %d want %d", res.SizeBytes, len(res.Gzip))
	}
	sum := sha256.Sum256(res.Gzip)
	wantSha := fmt.Sprintf("%x", sum[:])
	if res.SHA256Hex != wantSha {
		t.Fatalf("res.SHA256Hex = %s, want %s", res.SHA256Hex, wantSha)
	}
}

func assertNoHTMLEscape(t *testing.T, raw string) {
	t.Helper()
	if !strings.Contains(raw, "<b>bold</b>") {
		t.Errorf("raw JSON should contain unescaped HTML; raw=%s", raw)
	}
	if strings.Contains(raw, `\u003c`) {
		t.Errorf("raw JSON should not contain HTML-escaped sequences; raw=%s", raw)
	}
}

func TestBuildLightIndex_GzipInput_OK(t *testing.T) {
	item := `{"name":"a","full_name":"homebrew/core/a","aliases":[],
		"oldnames":[],"desc":"x","tap":"homebrew/core","homepage":"h",
		"license":"L","versions":{"stable":"1.0.0"},
		"deprecated":false,"deprecation_date":"","deprecation_reason":"",
		"disabled":false,"disable_date":"","disable_reason":"",
		"keg_only":false,"bottle":{"stable":{"files":{}}},"dependencies":[],
		"outdated":false,"pinned":false}`
	input := upstreamArrayJSON(item)
	gzInput := mustGzip(t, input)
	src := io.NopCloser(bytes.NewReader(gzInput))

	res, err := BuildLightIndex(context.Background(), src)
	if err != nil {
		t.Fatalf("BuildLightIndex (gz input): %v", err)
	}
	_, idx := decodeResultGzip(t, res.Gzip)
	if len(idx.Items) != 1 || idx.Items[0].Name != "a" {
		t.Fatalf("unexpected decoded items: %+v", idx.Items)
	}
}

func TestBuildLightIndex_EmptyArray(t *testing.T) {
	src := io.NopCloser(bytes.NewReader([]byte("[]")))
	res, err := BuildLightIndex(context.Background(), src)
	if err != nil {
		t.Fatalf("BuildLightIndex(empty): %v", err)
	}
	raw, idx := decodeResultGzip(t, res.Gzip)
	if len(idx.Items) != 0 {
		t.Fatalf("expected 0 items, got %d (raw=%s)", len(idx.Items), raw)
	}
	if res.Count != 0 {
		t.Fatalf("res.Count = %d, want 0", res.Count)
	}
}

func TestBuildLightIndex_InvalidTopLevel(t *testing.T) {
	// Not an array → should fail with "want array".
	src := io.NopCloser(bytes.NewReader([]byte(`{"not":"array"}`)))
	_, err := BuildLightIndex(context.Background(), src)
	if err == nil || !strings.Contains(err.Error(), "want array") {
		t.Fatalf("expected 'want array' error, got %v", err)
	}
}

func TestBuildLightIndex_ContextCanceled(t *testing.T) {
	// Provide one element so decoder enters the loop and hits ctx.Done().
	item := `{"name":"a","full_name":"homebrew/core/a","aliases":[],
		"oldnames":[],"desc":"","tap":"","homepage":"","license":"",
		"versions":{"stable":"1.0.0"},"bottle":{"stable":{"files":{}}},"dependencies":[]}`
	src := io.NopCloser(bytes.NewReader(upstreamArrayJSON(item)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling
	_, err := BuildLightIndex(ctx, src)
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
}

func TestBuildLightIndex_CloseErrorPropagates(t *testing.T) {
	item := `{"name":"a","full_name":"homebrew/core/a","aliases":[],
		"oldnames":[],"desc":"","tap":"","homepage":"","license":"",
		"versions":{"stable":"1.0.0"},"bottle":{"stable":{"files":{}}},"dependencies":[]}`
	src := rcWithCloseErr{
		Reader:   bytes.NewReader(upstreamArrayJSON(item)),
		closeErr: fmt.Errorf("boom"),
	}

	_, err := BuildLightIndex(context.Background(), src)
	// Build would succeed, then deferred Close should turn it into an error.
	if err == nil || !strings.Contains(err.Error(), "close failed") {
		t.Fatalf("expected close failed error, got %v", err)
	}
}

func TestBuildLightIndex_GeneratedTimestampPresent(t *testing.T) {
	// Sanity check that generated_at is set and schema is 1.
	item := `{"name":"ts","full_name":"homebrew/core/ts","aliases":[],
		"oldnames":[],"desc":"","tap":"","homepage":"","license":"",
		"versions":{"stable":"0.1.0"},"bottle":{"stable":{"files":{}}},"dependencies":[]}`
	src := io.NopCloser(bytes.NewReader(upstreamArrayJSON(item)))
	res, err := BuildLightIndex(context.Background(), src)
	if err != nil {
		t.Fatalf("BuildLightIndex: %v", err)
	}

	_, idx := decodeResultGzip(t, res.Gzip)
	if idx.Schema != SchemaVersion {
		t.Fatalf("schema=%d, want %d", idx.Schema, SchemaVersion)
	}
	// generated_at should be a valid time (non-zero)
	if idx.GeneratedAt.IsZero() {
		t.Fatalf("generated_at missing/zero")
	}
	// timestamp should be recent-ish
	if time.Since(idx.GeneratedAt) > time.Minute {
		t.Fatalf("generated_at too old: %s", idx.GeneratedAt)
	}
}
