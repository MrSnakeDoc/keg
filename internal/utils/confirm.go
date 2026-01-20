package utils

import (
	"fmt"
	"strings"

	"github.com/MrSnakeDoc/keg/internal/logger"
)

func ConfirmOrAbort(message, errormsg string) error {
	logger.WarnInline("%s", message)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil && err.Error() != "unexpected newline" {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(response)) != "y" {
		return fmt.Errorf("%s", errormsg)
	}

	return nil
}
