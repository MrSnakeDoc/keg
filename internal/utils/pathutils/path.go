package pathutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ToHomePathFormat(path string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home), nil
	}
	return path, nil
}

func ToAbsolutePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
	}
	return path, nil
}
