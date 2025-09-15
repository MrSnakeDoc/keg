package index

import "time"

// Schema version of the light index payload
const SchemaVersion = 1

// Light item we expose in the index.
type ItemLight struct {
	Name     string `json:"name"`
	FullName string `json:"full_name,omitempty"`
	Tap      string `json:"tap,omitempty"`
	Version  string `json:"version,omitempty"`
	Desc     string `json:"desc,omitempty"`
	Homepage string `json:"homepage,omitempty"`
	License  string `json:"license,omitempty"`

	Deprecated        bool   `json:"deprecated,omitempty"`
	DeprecationDate   string `json:"deprecation_date,omitempty"` // YYYY-MM-DD
	DeprecationReason string `json:"deprecation_reason,omitempty"`
	Replacement       string `json:"replacement,omitempty"` // formula or cask name

	Disabled      bool   `json:"disabled,omitempty"`
	DisableDate   string `json:"disable_date,omitempty"` // YYYY-MM-DD
	DisableReason string `json:"disable_reason,omitempty"`

	KegOnly   bool `json:"keg_only,omitempty"`
	HasBottle bool `json:"has_bottle,omitempty"`

	Aliases  []string `json:"aliases,omitempty"`
	OldNames []string `json:"oldnames,omitempty"`

	DepCount int `json:"dep_count,omitempty"`

	Outdated bool `json:"outdated,omitempty"`
	Pinned   bool `json:"pinned,omitempty"`
}

// Full index payload (before gzip)
type IndexLight struct {
	Schema      int         `json:"schema"`
	GeneratedAt time.Time   `json:"generated_at"`
	Count       int         `json:"count"`
	Items       []ItemLight `json:"items"`
}

// Result of the build: gzipped JSON + meta
type Result struct {
	Gzip      []byte
	Generated time.Time
	Count     int
	SHA256Hex string // sha256 of the gzipped payload (lower hex)
	SizeBytes int64  // len(Gzip)
}

// Homebrew formula subset (only fields we care about).
type formula struct {
	Name     string   `json:"name"`
	FullName string   `json:"full_name"`
	Aliases  []string `json:"aliases"`
	OldNames []string `json:"oldnames"`
	Desc     string   `json:"desc"`
	Tap      string   `json:"tap"`
	Homepage string   `json:"homepage"`
	License  string   `json:"license"`

	Versions struct {
		Stable string `json:"stable"`
	} `json:"versions"`

	// Deprecation info
	Deprecated                    bool   `json:"deprecated"`
	DeprecationDate               string `json:"deprecation_date"`
	DeprecationReason             string `json:"deprecation_reason"`
	DeprecationReplacementFormula string `json:"deprecation_replacement_formula"`
	DeprecationReplacementCask    string `json:"deprecation_replacement_cask"`

	// Disable info
	Disabled      bool   `json:"disabled"`
	DisableDate   string `json:"disable_date"`
	DisableReason string `json:"disable_reason"`

	// Keg-only
	KegOnly bool `json:"keg_only"`

	// Bottle info
	Bottle struct {
		Stable struct {
			Files map[string]struct {
				Cellar string `json:"cellar"`
				URL    string `json:"url"`
				Sha256 string `json:"sha256"`
			} `json:"files"`
		} `json:"stable"`
	} `json:"bottle"`

	// Dependencies
	Dependencies []string `json:"dependencies"`

	// Status flags
	Outdated bool `json:"outdated"`
	Pinned   bool `json:"pinned"`
}
