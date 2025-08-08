package internal

import (
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/remove"

	"github.com/spf13/cobra"
)

func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [packages...]",
		Short: "Remove packages from configuration",
		Long: `Remove packages from keg.yml configuration.

Examples:
  keg remove bat                     # Remove single package
  keg remove bat starship            # Remove multiple packages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}
			// Create remover
			r := remove.New(cfg, nil)

			return r.Execute(args)
		},
	}

	return cmd
}
