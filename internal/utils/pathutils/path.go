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

	relative, err := filepath.Rel(home, path)
	if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		if relative == "." {
			return "~", nil
		}
		return filepath.Join("~", relative), nil
	}
	return path, nil
}

func ToAbsolutePath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
	}
	return path, nil
}
