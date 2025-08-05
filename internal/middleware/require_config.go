package middleware

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/spf13/cobra"
)

func RequireConfig(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error {
	_, err := globalconfig.LoadPersistentConfig()
	if err != nil {
		return fmt.Errorf("missing config: %w", err)
	}
	return next(cmd, args)
}
