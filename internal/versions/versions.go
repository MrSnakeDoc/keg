package versions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MrSnakeDoc/keg/internal/runner"
)

type Info struct {
	Installed string    `json:"installed"`
	Latest    string    `json:"latest"`
	FetchedAt time.Time `json:"ts"`
}

type Resolver struct {
	Runner        runner.CommandRunner
	TTL           time.Duration
	MaxBatchSize  int // default 50
	GlobalTimeout time.Duration
	ChunkTimeout  time.Duration
}

const cacheFileName = "pkg_versions.json"

func NewResolver(r runner.CommandRunner) *Resolver {
	if r == nil {
		r = &runner.ExecRunner{}
	}
	return &Resolver{
		Runner:        r,
		TTL:           6 * time.Hour,
		MaxBatchSize:  50,
		GlobalTimeout: 15 * time.Second,
		ChunkTimeout:  5 * time.Second,
	}
}

func computeRefreshSet(cache map[string]Info, names []string, ttl time.Duration, now time.Time) []string {
	toRefresh := make([]string, 0, len(names))
	for _, n := range names {
		v, ok := cache[n]
		if !ok || v.FetchedAt.IsZero() || now.Sub(v.FetchedAt) > ttl {
			toRefresh = append(toRefresh, n)
		}
	}
	return toRefresh
}

func (rv *Resolver) refreshChunksParallel(ctx context.Context, chunks [][]string) (map[string]Info, []error) {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		all  = make(map[string]Info)
		errs []error
	)

	wg.Add(len(chunks))
	for i, chunk := range chunks {
		go func(_ int, chunk []string) {
			defer wg.Done()
			data, err := rv.resolveChunk(ctx, chunk)
			mu.Lock()
			if err != nil {
				errs = append(errs, err)
			}
			for k, v := range data {
				all[k] = v
			}
			mu.Unlock()
		}(i, chunk)
	}
	wg.Wait()
	return all, errs
}

func assembleOutput(names []string, cache map[string]Info) map[string]Info {
	out := make(map[string]Info, len(names))
	for _, n := range names {
		if v, ok := cache[n]; ok {
			out[n] = v
		} else {
			out[n] = Info{}
		}
	}
	return out
}

// ResolveBulk returns version info for all names.
// It uses cache (~/.local/state/keg/pkg_versions.json), refreshes expired/missing via
// `brew info --json=v2 <chunk...>` in parallel (chunks of 50), merges, and saves cache.
func (rv *Resolver) ResolveBulk(ctx context.Context, names []string) (map[string]Info, error) {
	names = dedupeAndSort(names)
	cache, _ := LoadCache() // best-effort

	now := time.Now()
	toRefresh := computeRefreshSet(cache, names, rv.TTL, now)
	if len(toRefresh) == 0 {
		return assembleOutput(names, cache), nil
	}

	ctx, cancel := context.WithTimeout(ctx, rv.GlobalTimeout)
	defer cancel()

	// chunk of 50 and refresh in parallel
	chunks := chunkStrings(toRefresh, max(1, rv.MaxBatchSize))

	refreshed, errs := rv.refreshChunksParallel(ctx, chunks)

	// merge results → cache
	for k, v := range refreshed {
		cache[k] = v
	}
	out := assembleOutput(names, cache)

	// save cache (best-effort)
	_ = SaveCache(cache)

	// if all chunks failed, we propagate a global error
	if len(errs) == len(chunks) && len(chunks) > 0 {
		return out, fmt.Errorf("failed to refresh versions for all chunks: %w", errors.Join(errs...))
	}
	return out, nil
}

// Touch updates the cache for a single package after an upgrade/install.
// Latest is set to Installed by default to avoid stale displays just after action.
func (rv *Resolver) Touch(name, newInstalled string) error {
	cache, _ := LoadCache()
	now := time.Now()
	cache[name] = Info{
		Installed: newInstalled,
		Latest:    newInstalled,
		FetchedAt: now,
	}
	return SaveCache(cache)
}

// VersionsCachePath returns ~/.local/state/keg/pkg_versions.json (XDG-state-like).
func VersionsCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Respect XDG_STATE_HOME if present, else ~/.local/state
	stateHome := os.Getenv("XDG_STATE_HOME")
	if strings.TrimSpace(stateHome) == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(stateHome, "keg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, cacheFileName), nil
}

func LoadCache() (map[string]Info, error) {
	path, err := VersionsCachePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Info{}, nil
		}
		return nil, err
	}
	var m map[string]Info
	if err := json.Unmarshal(b, &m); err != nil {
		// corrupt -> start clean
		return map[string]Info{}, nil
	}
	return m, nil
}

func SaveCache(m map[string]Info) error {
	path, err := VersionsCachePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (rv *Resolver) Remove(name string) error {
	cache, _ := LoadCache()
	delete(cache, name)
	return SaveCache(cache)
}

// -------- Chunk resolution (brew info --json=v2) --------

func (rv *Resolver) resolveChunk(ctx context.Context, names []string) (map[string]Info, error) {
	// Defensive: empty chunk
	if len(names) == 0 {
		return map[string]Info{}, nil
	}
	// Build command
	args := append([]string{"info", "--json=v2"}, names...)
	// Use explicit runner.ModeStdout for clarity and to avoid zero-value assumptions.
	var mode runner.Mode
	chCtx, cancel := context.WithTimeout(ctx, rv.ChunkTimeout)
	defer cancel()

	out, err := rv.Runner.Run(chCtx, rv.ChunkTimeout, mode, "brew", args...)
	if err != nil {
		return nil, fmt.Errorf("brew info failed for chunk (%d pkgs): %w", len(names), err)
	}

	parsed, err := parseBrewInfoJSON(out)
	if err != nil {
		return nil, fmt.Errorf("failed to parse brew info json: %w", err)
	}
	now := time.Now()
	res := make(map[string]Info, len(parsed))
	for name, pi := range parsed {
		installed := ""
		if len(pi.Installed) > 0 {
			installed = pi.Installed[0].Version
		}
		latest := pi.Versions.Stable
		res[name] = Info{
			Installed: installed,
			Latest:    latest,
			FetchedAt: now,
		}
	}
	// Ensure every requested name is present, even if missing in the JSON (unknown package)
	for _, n := range names {
		if _, ok := res[n]; !ok {
			res[n] = Info{FetchedAt: now}
		}
	}
	return res, nil
}

// -------- JSON parsing (minimal schema) --------

type brewInfo struct {
	Formulae []brewFormula `json:"formulae"`
}

type brewFormula struct {
	Name      string          `json:"name"`
	Versions  brewVersions    `json:"versions"`
	Installed []brewInstalled `json:"installed"`
	// many fields omitted intentionally
}

type brewVersions struct {
	Stable string `json:"stable"`
	// head/bottle omitted
}

type brewInstalled struct {
	Version string `json:"version"`
}

func parseBrewInfoJSON(b []byte) (map[string]brewFormula, error) {
	var bi brewInfo
	if err := json.Unmarshal(b, &bi); err != nil {
		return nil, err
	}
	out := make(map[string]brewFormula, len(bi.Formulae))
	for _, f := range bi.Formulae {
		out[f.Name] = f
	}
	return out, nil
}

// -------- Helpers --------

func chunkStrings(in []string, size int) [][]string {
	if size <= 0 || len(in) == 0 {
		return [][]string{in}
	}
	var chunks [][]string
	for i := 0; i < len(in); i += size {
		end := i + size
		if end > len(in) {
			end = len(in)
		}
		chunks = append(chunks, in[i:end])
	}
	return chunks
}

func dedupeAndSort(in []string) []string {
	if len(in) == 0 {
		return in
	}
	set := make(map[string]struct{}, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			set[s] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
