package scheduler

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/MrSnakeDoc/keg/internal/index"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/service"
	"github.com/MrSnakeDoc/keg/internal/store"
)

// --- Test setup ---

func TestMain(m *testing.M) {
	logger.UseTestMode()
	os.Exit(m.Run())
}

// mockClient simulates AdvancedHTTPClient
type mockClient struct {
	res   service.FetchResult
	err   error
	calls int
}

func (m *mockClient) FetchWithETag(ctx context.Context, url, prevETag string, maxBytes int64) (service.FetchResult, error) {
	m.calls++
	return m.res, m.err
}

// helper to create a gzipped fake index
func makeFakeFormulaArray(t *testing.T, items []index.ItemLight) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(items); err != nil {
		t.Fatalf("encode fake array: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
	return buf.Bytes()
}

// helper to create a test FS
func newTestFS(t *testing.T) *store.FS {
	t.Helper()
	tmp := t.TempDir()
	fs, err := store.NewFS(tmp)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	return fs
}

// --- Tests ---

func TestRefreshIndex_SkipFresh(t *testing.T) {
	fs := newTestFS(t)

	// Write index with recent LastChecked
	data := makeFakeFormulaArray(t, []index.ItemLight{{Name: "foo"}})
	meta := store.Meta{LastChecked: time.Now().UTC(), GeneratedAt: time.Now().UTC(), SizeBytes: int64(len(data))}
	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(data), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}

	client := &mockClient{}
	err := RefreshIndex(context.Background(), fs, (*service.AdvancedHTTPClient)(nil), false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if client.calls != 0 {
		t.Errorf("expected no fetch, got %d", client.calls)
	}
}

func TestRefreshIndex_ForceRefresh(t *testing.T) {
	fs := newTestFS(t)

	// Write stale index
	data := makeFakeFormulaArray(t, []index.ItemLight{{Name: "bar"}})
	meta := store.Meta{LastChecked: time.Now().Add(-48 * time.Hour), GeneratedAt: time.Now().Add(-48 * time.Hour), SizeBytes: int64(len(data))}
	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(data), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}

	newData := makeFakeFormulaArray(t, []index.ItemLight{{Name: "baz"}})
	client := &mockClient{
		res: service.FetchResult{
			Status: 200,
			Body:   io.NopCloser(bytes.NewReader(newData)),
			ETag:   "etag-123",
		},
	}

	err := RefreshIndex(context.Background(), fs, client, true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if client.calls != 1 {
		t.Errorf("expected 1 fetch, got %d", client.calls)
	}
}

func TestRefreshIndex_304WithIndex(t *testing.T) {
	fs := newTestFS(t)

	// Write index so "present=true"
	data := makeFakeFormulaArray(t, []index.ItemLight{{Name: "zap"}})
	meta := store.Meta{LastChecked: time.Now().Add(-48 * time.Hour), GeneratedAt: time.Now().Add(-48 * time.Hour), SizeBytes: int64(len(data)), UpstreamETag: "etag-old"}
	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(data), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}

	client := &mockClient{
		res: service.FetchResult{
			Status: 304,
			ETag:   "etag-old",
		},
	}

	err := RefreshIndex(context.Background(), fs, client, false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if client.calls != 1 {
		t.Errorf("expected 1 fetch, got %d", client.calls)
	}
}

func TestRefreshIndex_304NoIndexRefetch(t *testing.T) {
	fs := newTestFS(t)

	// No index, but meta present
	meta := store.Meta{LastChecked: time.Now().Add(-48 * time.Hour), UpstreamETag: "etag-dangling"}
	if err := fs.WriteMeta(context.Background(), meta); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}
	fs.ClearHotCacheForTest()

	// First call returns 304
	client := &mockClient{
		res: service.FetchResult{
			Status: 304,
			ETag:   "etag-dangling",
		},
	}

	// Force next call to be 200
	newData := makeFakeFormulaArray(t, []index.ItemLight{{Name: "recovered"}})
	client2 := &mockClient{
		res: service.FetchResult{
			Status: 200,
			Body:   io.NopCloser(bytes.NewReader(newData)),
			ETag:   "etag-new",
		},
	}

	// Replace client in second phase
	err := RefreshIndex(context.Background(), fs, client, false)
	if err == nil {
		t.Fatalf("expected error because dangling 304 needs refetch, got nil")
	}

	err = RefreshIndex(context.Background(), fs, client2, false)
	if err != nil {
		t.Fatalf("unexpected err after refetch: %v", err)
	}
}
