package utils

import (
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
