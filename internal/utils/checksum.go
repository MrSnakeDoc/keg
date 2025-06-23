package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// ChecksumVerifiedReader returns a new reader if SHA256 checksum matches.
func ChecksumVerifiedReader(r io.Reader, expected string) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	actual := sha256Sum(data)
	if actual != expected {
		return nil, fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return bytes.NewReader(data), nil
}

// sha256Sum returns the SHA256 hash of the given data as a hex string.
func sha256Sum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
