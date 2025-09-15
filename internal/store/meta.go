package store

import "time"

type Meta struct {
	ETag        string    `json:"etag"`
	GeneratedAt time.Time `json:"generated_at"`
	Count       int       `json:"count"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256      string    `json:"sha256"`

	// For upstream fetch:
	UpstreamETag string `json:"upstream_etag,omitempty"`

	LastSuccess time.Time `json:"last_success"`
	LastChecked time.Time `json:"last_checked"`
}
