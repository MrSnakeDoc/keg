package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

type FS struct {
	dir       string
	indexPath string
	metaPath  string
	mu        sync.RWMutex
	hotData   []byte
	hotETag   string
	hotGenAt  time.Time
	hotSize   int64
}

type Store interface {
	// Hot path from RAM (may be nil if not loaded yet)
	GetHot() (data []byte, etag string, generatedAt time.Time, size int64)

	// Fallback: open gz file for streaming
	OpenIndexGZ(ctx context.Context) (rc io.ReadSeekCloser, etag string, generatedAt time.Time, size int64, err error)

	// WriteIndexGZ writes the gzipped index atomically and updates meta + hot cache.
	WriteIndexGZ(ctx context.Context, r io.Reader, meta Meta) error

	// ReadMeta reads meta.json (if present).
	ReadMeta(ctx context.Context) (Meta, error)
}

func NewFS(dataDir string) (*FS, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dataDir, err)
	}
	s := &FS{
		dir:       dataDir,
		indexPath: filepath.Join(dataDir, "index-light.json.gz"),
		metaPath:  filepath.Join(dataDir, "meta.json"),
	}
	_ = s.loadHotFromDisk() // best-effort at boot
	return s, nil
}

// GetHot returns a snapshot of the in-memory gzip (if loaded).
func (s *FS) GetHot() (data []byte, etag string, generatedAt time.Time, size int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.hotData == nil {
		return nil, "", time.Time{}, 0
	}
	return append([]byte(nil), s.hotData...), s.hotETag, s.hotGenAt, s.hotSize
}

// OpenIndexGZ opens the file for streaming (fallback when hot cache is empty).
func (s *FS) OpenIndexGZ(ctx context.Context) (rc io.ReadSeekCloser, etag string, generatedAt time.Time, size int64, err error) {
	// fast-path: hot cache
	if data, e, g, sz := s.GetHot(); data != nil {
		return utils.NewBytesReadSeekCloser(data), e, g, sz, nil
	}

	f, err := os.Open(s.indexPath)
	if err != nil {
		return nil, "", time.Time{}, 0, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, "", time.Time{}, 0, err
	}

	m, _ := s.ReadMeta(ctx) // best-effort
	return f, m.ETag, m.GeneratedAt, fi.Size(), nil
}

// WriteIndexGZ writes the gzipped index atomically and updates meta + hot cache.
// r must be the COMPLETE gzipped payload to persist as-is.
// meta must contain at least ETag, GeneratedAt, SizeBytes (count/sha256 optional).
func (s *FS) WriteIndexGZ(ctx context.Context, r io.Reader, meta Meta) error {
	logger.Debug("writing index.gz to %s (size=%s)", s.indexPath, utils.HumanSize(meta.SizeBytes))

	tmp := s.indexPath + ".tmp"
	if err := utils.WriteFileAtomic(tmp, s.indexPath, r); err != nil {
		return err
	}
	// Write meta.json (atomic as well)
	if err := utils.WriteJSONAtomic(s.metaPath, meta); err != nil {
		return err
	}

	// Reload hot cache from disk (cheap; OS page cache helps)
	return s.loadHotFromDisk()
}

// ReadMeta reads meta.json (if present).
func (s *FS) ReadMeta(ctx context.Context) (met Meta, err error) {
	f, err := os.Open(s.metaPath)
	if err != nil {
		return Meta{}, err
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	var m Meta
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return Meta{}, err
	}
	return m, err
}

func (s *FS) WriteMeta(ctx context.Context, m Meta) error {
	return utils.WriteJSONAtomic(s.metaPath, m)
}

// --- internals ---

func (s *FS) loadHotFromDisk() error {
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		// Not ready yet is fine
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	m, err := s.ReadMeta(context.Background())
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.hotData = data
	s.hotETag = m.ETag
	s.hotGenAt = m.GeneratedAt
	s.hotSize = int64(len(data))
	s.mu.Unlock()
	return nil
}

// clearHotCacheForTest is only used in unit tests to force a disk fallback
// in OpenIndexGZ. It resets the in-memory hot cache to an empty state.
func (s *FS) ClearHotCacheForTest() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hotData = nil
	s.hotETag = ""
	s.hotGenAt = time.Time{}
	s.hotSize = 0
}
