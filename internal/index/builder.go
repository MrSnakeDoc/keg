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
	"time"

	"github.com/MrSnakeDoc/keg/internal/utils"
)

// BuildLightIndex builds the light index from Homebrew's formula JSON, which may be gzip-compressed.
// It returns a gzipped JSON payload and associated metadata.
func BuildLightIndex(ctx context.Context, src io.ReadCloser) (res Result, err error) {
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	rc, err := utils.MaybeGunzip(src)
	if err != nil {
		return Result{}, fmt.Errorf("gunzip: %w", err)
	}

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	enc := json.NewEncoder(gw)
	enc.SetEscapeHTML(false)

	now := time.Now().UTC()

	// parse input array
	dec := json.NewDecoder(rc)
	tok, err := dec.Token()
	if err != nil {
		return Result{}, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return Result{}, fmt.Errorf("want array")
	}

	// open items array
	if _, err := gw.Write([]byte(`{"schema":1,"generated_at":"` + now.Format(time.RFC3339) + `","count":0,"items":[`)); err != nil {
		return Result{}, err
	}
	first := true
	var count int
	for dec.More() {
		select {
		case <-ctx.Done():
			return Result{}, ctx.Err()
		default:
		}
		var f formula
		if err := dec.Decode(&f); err != nil {
			return Result{}, err
		}
		it := toItemLight(f)
		if !first {
			if _, err := gw.Write([]byte(",")); err != nil {
				return Result{}, err
			}
		}
		first = false
		if err := enc.Encode(it); err != nil {
			return Result{}, err
		}
		count++
	}
	if _, err := gw.Write([]byte(`]}`)); err != nil {
		return Result{}, err
	}
	if err := gw.Close(); err != nil {
		return Result{}, fmt.Errorf("closing gzip writer: %w", err)
	}

	sum := sha256.Sum256(gzBuf.Bytes())
	return Result{
		Gzip:      gzBuf.Bytes(),
		Generated: now,
		Count:     count,
		SHA256Hex: fmt.Sprintf("%x", sum[:]),
		SizeBytes: int64(gzBuf.Len()),
	}, err
}

// ----- internals -----

func toItemLight(f formula) ItemLight {
	// Build search text (lowercase, spaces)
	var sb strings.Builder
	sb.Grow(len(f.Name) + len(f.Desc) + 32)
	sb.WriteString(strings.ToLower(f.Name))
	if len(f.Aliases) > 0 {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(strings.Join(f.Aliases, " ")))
	}
	if len(f.OldNames) > 0 {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(strings.Join(f.OldNames, " ")))
	}
	if f.Desc != "" {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(f.Desc))
	}

	return ItemLight{
		Name:     f.Name,
		FullName: f.FullName,
		Tap:      f.Tap,
		Version:  f.Versions.Stable,
		Desc:     f.Desc,
		Homepage: f.Homepage,
		License:  f.License,

		Deprecated:        f.Deprecated,
		DeprecationDate:   f.DeprecationDate,
		DeprecationReason: f.DeprecationReason,
		Replacement: func() string {
			if f.DeprecationReplacementFormula != "" {
				return f.DeprecationReplacementFormula
			}
			if f.DeprecationReplacementCask != "" {
				return f.DeprecationReplacementCask
			}
			return ""
		}(),

		Disabled:      f.Disabled,
		DisableDate:   f.DisableDate,
		DisableReason: f.DisableReason,

		KegOnly:   f.KegOnly,
		HasBottle: len(f.Bottle.Stable.Files) > 0,

		Aliases:  f.Aliases,
		OldNames: f.OldNames,

		DepCount: len(f.Dependencies),

		Outdated: f.Outdated,
		Pinned:   f.Pinned,
	}
}
