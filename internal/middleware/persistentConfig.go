package middleware

import (
	"context"
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/globalconfig"
	"github.com/spf13/cobra"
)

func RequireConfig(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error {
	pconf, err := globalconfig.LoadPersistentConfig()
	if err != nil {
		return fmt.Errorf("missing config: %w", err)
	}

	ctx := context.WithValue(cmd.Context(), CtxKeyPConfig, pconf)
	cmd.SetContext(ctx)

	return next(cmd, args)
}
