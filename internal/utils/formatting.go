package utils

import "regexp"

func StripANSI(input string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(input, "")
}

func GetMaxWidth(lines []string) int {
	maxWidth := 0
	for _, line := range lines {
		length := len(StripANSI(line))
		if length > maxWidth {
			maxWidth = length
		}
	}
	return maxWidth
}
