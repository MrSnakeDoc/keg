package store

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func writeGzPayload(t *testing.T, items any) *bytes.Buffer {
	t.Helper()
	var raw bytes.Buffer
	gzw := gzip.NewWriter(&raw)
	if err := json.NewEncoder(gzw).Encode(items); err != nil {
		t.Fatalf("encode gz: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gz: %v", err)
	}
	return &raw
}

func newTestFS(t *testing.T) *FS {
	t.Helper()
	tmp := t.TempDir()
	fs, err := NewFS(tmp)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	return fs
}

func TestWriteAndOpenIndexGZ_Roundtrip(t *testing.T) {
	fs := newTestFS(t)
	payload := map[string]string{"foo": "bar"}
	raw := writeGzPayload(t, payload)

	meta := Meta{
		ETag:        "etag-123",
		GeneratedAt: time.Now().UTC(),
		SizeBytes:   int64(raw.Len()),
	}

	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(raw.Bytes()), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}

	// Open again
	rc, etag, genAt, sz, err := fs.OpenIndexGZ(context.Background())
	if err != nil {
		t.Fatalf("OpenIndexGZ: %v", err)
	}
	defer func() {
		if cerr := rc.Close(); cerr != nil {
			t.Errorf("close: %v", cerr)
		}
	}()

	if etag != "etag-123" {
		t.Errorf("wrong etag: %s", etag)
	}
	if sz != int64(raw.Len()) {
		t.Errorf("wrong size: got %d, want %d", sz, raw.Len())
	}
	if genAt.IsZero() {
		t.Errorf("expected non-zero genAt")
	}
}

func TestGetHot_AfterWrite(t *testing.T) {
	fs := newTestFS(t)
	raw := writeGzPayload(t, map[string]string{"hello": "world"})
	meta := Meta{ETag: "etag-xxx", GeneratedAt: time.Now().UTC(), SizeBytes: int64(raw.Len())}

	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(raw.Bytes()), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}
	_, etag, genAt, sz := fs.GetHot()
	if etag != "etag-xxx" || sz == 0 || genAt.IsZero() {
		t.Errorf("hot cache not populated correctly: etag=%s sz=%d genAt=%v", etag, sz, genAt)
	}
}

func TestReadMetaAndWriteMeta(t *testing.T) {
	fs := newTestFS(t)
	meta := Meta{ETag: "etag-yolo", GeneratedAt: time.Now().UTC(), SizeBytes: 42}
	if err := fs.WriteMeta(context.Background(), meta); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}
	got, err := fs.ReadMeta(context.Background())
	if err != nil {
		t.Fatalf("ReadMeta: %v", err)
	}
	if got.ETag != "etag-yolo" {
		t.Errorf("expected etag-yolo, got %s", got.ETag)
	}
	if got.SizeBytes != 42 {
		t.Errorf("expected 42, got %d", got.SizeBytes)
	}
}

type badReader struct{}

func (b *badReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestWriteIndexGZ_ErrorPropagation(t *testing.T) {
	fs := newTestFS(t)
	meta := Meta{ETag: "fail", GeneratedAt: time.Now().UTC(), SizeBytes: 99}
	err := fs.WriteIndexGZ(context.Background(), &badReader{}, meta)
	if err == nil {
		t.Fatalf("expected error from bad reader, got nil")
	}
}

func TestOpenIndexGZ_FallbackToDisk(t *testing.T) {
	fs := newTestFS(t)

	data := []byte("compressed-data")
	meta := Meta{
		ETag:        "etag-disk",
		GeneratedAt: time.Now().UTC(),
		SizeBytes:   int64(len(data)),
	}
	if err := fs.WriteIndexGZ(context.Background(), bytes.NewReader(data), meta); err != nil {
		t.Fatalf("WriteIndexGZ: %v", err)
	}

	fs.ClearHotCacheForTest()

	rc, etag, genAt, size, err := fs.OpenIndexGZ(context.Background())
	if err != nil {
		t.Fatalf("OpenIndexGZ fallback: %v", err)
	}

	defer func() {
		if cerr := rc.Close(); cerr != nil {
			t.Errorf("close: %v", cerr)
		}
	}()

	if etag != "etag-disk" {
		t.Errorf("expected etag-disk, got %s", etag)
	}
	if size != int64(len(data)) {
		t.Errorf("expected size %d, got %d", len(data), size)
	}
	if genAt.IsZero() {
		t.Errorf("expected non-zero GeneratedAt")
	}

	b, _ := io.ReadAll(rc)
	if string(b) != string(data) {
		t.Errorf("expected data %q, got %q", data, b)
	}
}
