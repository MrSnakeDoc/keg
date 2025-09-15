package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/logger"
)

type VersionInfo struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// IsNewerVersion compares two semantic versions and returns true if remote > local.
func IsNewerVersion(remote, local string) (bool, error) {
	if !IsSemver(remote) || !IsSemver(local) {
		return false, errors.New("invalid semantic version format (expected x.y.z)")
	}

	rParts := strings.Split(remote, ".")
	lParts := strings.Split(local, ".")

	for i := 0; i < 3; i++ {
		rNum, _ := strconv.Atoi(rParts[i])
		lNum, _ := strconv.Atoi(lParts[i])

		switch {
		case rNum > lNum:
			return true, nil
		case rNum < lNum:
			return false, nil
		}
	}

	return false, nil // same version
}

// IsSemver returns true if the string is a valid semver (x.y.z).
func IsSemver(v string) bool {
	return semverPattern.MatchString(v)
}

func AssetName(version string) string {
	return fmt.Sprintf("keg_%s_%s_%s", version, runtime.GOOS, runtime.GOARCH)
}

func ParseChecksumsForBinary(body, tag string) (string, error) {
	version := strings.TrimPrefix(tag, "v")
	target := AssetName(version)
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == target {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", target)
}

func ValidateVersion(info *VersionInfo) error {
	const (
		minVersionLength = 5
		minSHA256Length  = 64
		baseUpdateURL    = "https://github.com/MrSnakeDoc/keg/releases/download/"
	)

	if info.Version == "" || len(info.Version) < minVersionLength {

		logger.Debug("invalid version format")
		return nil
	}

	if info.SHA256 == "" || len(info.SHA256) != minSHA256Length {
		logger.Debug("invalid SHA256 format")
		return nil
	}

	if info.URL == "" || !strings.HasPrefix(info.URL, baseUpdateURL) {
		logger.Debug("invalid download URL: must start with %s", baseUpdateURL)
		return nil
	}

	return nil
}

// ValidateSHA256Checksum verifies if the SHA256 hash of the file matches the expected checksum.
func ValidateSHA256Checksum(filePath, expectedChecksum string) (err error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}

	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close failed: %w", cerr)
		}
	}()

	// Initialize SHA256 hasher
	hasher := sha256.New()

	// Copy the file content to the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute SHA256 for %s: %w", filePath, err)
	}

	// Compute the final hash
	computedHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare with the expected hash
	if computedHash != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, computedHash)
	}

	return err
}
