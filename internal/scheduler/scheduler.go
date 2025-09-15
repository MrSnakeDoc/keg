package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/index"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/service"
	"github.com/MrSnakeDoc/keg/internal/store"
	"github.com/MrSnakeDoc/keg/internal/utils"
)

// RefreshIndex refreshes the local Homebrew index if needed.
// - Skips if a fresh index (<24h) is already present.
// - Uses ETag to avoid re-downloading when not modified.
// - Persists the gzipped index and meta.json atomically.
func RefreshIndex(ctx context.Context, st store.Store, client service.AdvancedFetcher, refresh bool) error {
	now := time.Now().UTC()

	// Check local presence
	data, _, _, _ := st.GetHot()
	present := data != nil
	meta, _ := st.ReadMeta(ctx)

	// 24h gate
	last := meta.LastChecked
	if meta.LastSuccess.After(last) {
		last = meta.LastSuccess
	}
	if present && !last.IsZero() && now.Sub(last) < globalconfig.RefreshInterval && !refresh {
		logger.Debug("refresh: skip (last=%s, age=%s < %s)",
			last.Format(time.RFC3339), now.Sub(last).Truncate(time.Second), globalconfig.RefreshInterval)
		return nil
	}

	// Choose ETag
	prevETag := ""
	if present && meta.UpstreamETag != "" {
		prevETag = meta.UpstreamETag
	}

	// Fetch
	res, err := client.FetchWithETag(ctx, globalconfig.BrewFormulaURL, prevETag, int64(globalconfig.MaxDownloadBytes))
	if err != nil {
		updateMetaLastChecked(ctx, st, now)
		return fmt.Errorf("fetch: %w", err)
	}

	switch res.Status {
	case http.StatusNotModified:
		// Valid only if index is present
		updateMetaLastChecked(ctx, st, now)
		logger.Info("refresh: 304 Not Modified (upstream etag=%q)", prevETag)
		if present {
			return nil
		}

		// Edge case: got 304 but no local index → force refetch
		logger.Warn("dangling meta (no index.gz but 304); refetching without ETag")
		res2, err := client.FetchWithETag(ctx, globalconfig.BrewFormulaURL, "", int64(globalconfig.MaxDownloadBytes))
		if err != nil {
			return fmt.Errorf("refetch without etag: %w", err)
		}
		if res2.Status != http.StatusOK {
			return fmt.Errorf("unexpected status %d in refetch", res2.Status)
		}
		return persistBuild(ctx, st, res2, now)

	case http.StatusOK:
		return persistBuild(ctx, st, res, now)

	default:
		updateMetaLastChecked(ctx, st, now)
		return fmt.Errorf("unexpected status %d", res.Status)
	}
}

func persistBuild(ctx context.Context, st store.Store, res service.FetchResult, now time.Time) (err error) {
	defer func() {
		if cerr := res.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	start := time.Now()

	build, err := index.BuildLightIndex(ctx, res.Body)
	if err != nil {
		return fmt.Errorf("build index: %w", err)
	}

	logger.Debug("refresh: build took %s", time.Since(start).Truncate(time.Millisecond))

	newMeta := store.Meta{
		ETag:         "sha256:" + build.SHA256Hex,
		GeneratedAt:  build.Generated,
		Count:        build.Count,
		SizeBytes:    build.SizeBytes,
		SHA256:       build.SHA256Hex,
		UpstreamETag: res.ETag,
		LastSuccess:  now,
		LastChecked:  now,
	}
	if err := st.WriteIndexGZ(ctx, bytes.NewReader(build.Gzip), newMeta); err != nil {
		return fmt.Errorf("write store: %w", err)
	}
	logger.Debug("refresh: 200 OK → wrote index (items=%d, size=%s, etag=%s, upstream=%s)",
		build.Count, utils.HumanSize(newMeta.SizeBytes), newMeta.ETag, newMeta.UpstreamETag)
	return err
}

func updateMetaLastChecked(ctx context.Context, st store.Store, ts time.Time) {
	if mw, ok := any(st).(interface {
		WriteMeta(context.Context, store.Meta) error
	}); ok {
		m, err := st.ReadMeta(ctx)
		if err != nil {
			return
		}
		m.LastChecked = ts
		_ = mw.WriteMeta(ctx, m)
	}
}
