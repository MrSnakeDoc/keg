package middleware

import (
	"context"

	"github.com/MrSnakeDoc/keg/internal/utils"
	"github.com/spf13/cobra"
)

func LoadPkgList(cmd *cobra.Command, args []string, next func(cmd *cobra.Command, args []string) error) error {
	cfg, err := utils.LoadConfig()
	if err != nil {
		return err
	}

	ctx := context.WithValue(cmd.Context(), CtxKeyConfig, cfg)
	cmd.SetContext(ctx)

	return next(cmd, args)
}
