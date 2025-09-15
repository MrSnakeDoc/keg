package utils

import (
	"fmt"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/config"
)

func TransformToMap[T any](lines []string, transform func(string) (string, T)) map[string]T {
	result := make(map[string]T, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			k, v := transform(trimmed)
			result[k] = v
		}
	}
	return result
}

func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func NewUpdateState(base config.UpdateState, isNewer bool, version string) config.UpdateState {
	s := base
	s.UpdateAvailable = isNewer
	s.LatestVersion = version
	return s
}

func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}
