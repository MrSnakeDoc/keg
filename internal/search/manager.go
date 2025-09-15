package search

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/checker"
	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/MrSnakeDoc/keg/internal/index"
	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/MrSnakeDoc/keg/internal/scheduler"
	"github.com/MrSnakeDoc/keg/internal/service"
	"github.com/MrSnakeDoc/keg/internal/store"
	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/MrSnakeDoc/keg/internal/versions"
)

type Searcher struct {
	store  *store.FS
	client *service.AdvancedHTTPClient
}

func New(str *store.FS, client *service.AdvancedHTTPClient) *Searcher {
	if str == nil {
		fsStore, err := store.NewFS(globalconfig.GetConfigDir(globalconfig.DataDir))
		if err != nil {
			panic(fmt.Sprintf("failed to initialize FS store: %v", err))
		}
		str = fsStore
	}

	if client == nil {
		client = service.NewAdvancedHTTPClient("keg/" + checker.Version)
	}

	return &Searcher{
		store:  str,
		client: client,
	}
}

// ---- Orchestrator ----

func (s *Searcher) Execute(args []string, ctx context.Context, cfg *models.Config,
	exact, noDesc, regex, fzf, jsonOut bool, limit int, refresh bool, test bool,
) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !test {
		if err = scheduler.RefreshIndex(ctx, s.store, s.client, refresh); err != nil {
			logger.Warn("refresh failed: %v", err)
		}
	}

	// Open local gz index
	rc, _, _, _, err := s.store.OpenIndexGZ(ctx)
	if err != nil {
		return fmt.Errorf("open index: %w", err)
	}

	defer func() {
		if cerr := rc.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	// Gunzip
	gz, err := gzip.NewReader(rc)
	if err != nil {
		return fmt.Errorf("gunzip: %w", err)
	}

	defer func() {
		if cerr := gz.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	// Decode JSON index
	var idx index.IndexLight
	if err := json.NewDecoder(gz).Decode(&idx); err != nil {
		return fmt.Errorf("decode index: %w", err)
	}

	// Build search options
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	if regex {
		if _, err := regexp.Compile(query); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}

	searchOpts := SearchOptions{
		Exact: exact,
		Regex: regex,
		Desc:  !noDesc,
		Query: query,
	}

	outputOpts := OutputOptions{
		JSON:   jsonOut,
		FZF:    fzf,
		NoDesc: noDesc,
	}

	// Pipeline
	results := searchItems(idx.Items, searchOpts)

	results = limitItems(results, limit)

	return chosenOutput(cfg, results, outputOpts, fzf, jsonOut)
}

func chosenOutput(cfg *models.Config, items []index.ItemLight, opts OutputOptions, fzf bool, jsonOut bool) error {
	if jsonOut && fzf {
		return fmt.Errorf("cannot use --json and --fzf together")
	}

	if fzf {
		return outputFzfItems(items, opts)
	}

	if jsonOut {
		return outputJsonItems(items, opts)
	}

	return outputItems(cfg, items)
}

// ---- Options structs ----

type SearchOptions struct {
	Exact bool
	Regex bool
	Desc  bool
	Query string
}

type OutputOptions struct {
	JSON   bool
	FZF    bool
	NoDesc bool
}

// ---- Core functions ----

func searchItems(items []index.ItemLight, opts SearchOptions) []index.ItemLight {
	// no query = return all
	if opts.Query == "" {
		return items
	}
	query := strings.ToLower(opts.Query)

	var results []index.ItemLight

	for _, it := range items {
		if matchItem(it, query, opts) {
			results = append(results, it)
		}
	}
	return results
}

func matchItem(it index.ItemLight, query string, opts SearchOptions) bool {
	if opts.Exact {
		return matchExact(it, query)
	}
	if opts.Regex {
		return matchRegex(it, query, opts.Desc)
	}
	return matchSubstring(it, query, opts.Desc)
}

func matchExact(it index.ItemLight, query string) bool {
	if strings.EqualFold(it.Name, query) {
		return true
	}
	for _, v := range append(it.Aliases, it.OldNames...) {
		if strings.EqualFold(v, query) {
			return true
		}
	}
	return false
}

func matchRegex(it index.ItemLight, query string, desc bool) bool {
	re, err := regexp.Compile(query)
	if err != nil {
		return false
	}
	if re.MatchString(it.Name) {
		return true
	}
	for _, v := range append(it.Aliases, it.OldNames...) {
		if re.MatchString(v) {
			return true
		}
	}
	return desc && re.MatchString(it.Desc)
}

func matchSubstring(it index.ItemLight, query string, desc bool) bool {
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(it.Name), q) {
		return true
	}
	for _, v := range append(it.Aliases, it.OldNames...) {
		if strings.Contains(strings.ToLower(v), q) {
			return true
		}
	}
	return desc && strings.Contains(strings.ToLower(it.Desc), q)
}

func limitItems(items []index.ItemLight, limit int) []index.ItemLight {
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}

type searchRow struct {
	Name    string
	Status  string
	Desc    string
	SortKey string
}

func outputItems(cfg *models.Config, items []index.ItemLight) error {
	p := printer.NewColorPrinter()
	table := logger.CreateTable([]string{"Package", "Status", "Description"})

	// Load versions cache (safe even if missing)
	cache, _ := versions.LoadCache()

	// Build rows
	rows := utils.Map(items, func(it index.ItemLight) searchRow {
		status := p.Warning("-")
		if pkg, ok := findInConfig(cfg, it.Name); ok {
			// check if we have info in cache
			if v, ok := cache[pkg.Command]; ok && v.Installed != "" {
				status = p.Success("✓ installed")
			} else {
				status = p.Info("✗ missing")
			}
		}
		return searchRow{
			Name:    it.Name,
			Status:  status,
			Desc:    it.Desc,
			SortKey: strings.ToLower(it.Name),
		}
	})

	// Render
	for _, r := range rows {
		if err := table.Append([]string{r.Name, r.Status, r.Desc}); err != nil {
			return fmt.Errorf("append row: %w", err)
		}
	}
	if err := table.Render(); err != nil {
		return fmt.Errorf("render table: %w", err)
	}
	return nil
}

// findInConfig checks if a package is in cfg.Packages
func findInConfig(cfg *models.Config, name string) (*models.Package, bool) {
	for _, p := range cfg.Packages {
		if p.Command == name || (p.Binary != "" && p.Binary == name) {
			return &p, true
		}
	}
	return nil, false
}

func outputFzfItems(items []index.ItemLight, opts OutputOptions) error {
	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}
	if opts.FZF {
		for _, it := range items {
			fmt.Printf("%s\t%s\t%s\n", it.Name, strings.Join(it.Aliases, ","), it.Desc)
		}
		return nil
	}
	for _, it := range items {
		if opts.NoDesc {
			fmt.Println(it.Name)
		} else {
			fmt.Printf("%-20s %s\n", it.Name, it.Desc)
		}
	}
	return nil
}

func outputJsonItems(items []index.ItemLight, _ OutputOptions) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ") // pretty print
	return enc.Encode(items)
}
