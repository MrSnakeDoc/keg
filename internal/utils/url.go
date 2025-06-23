package utils

import (
	"fmt"
	"net/url"
)

func ParseSecureURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("insecure URL rejected")
	}
	return parsed, nil
}
