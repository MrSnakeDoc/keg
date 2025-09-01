package internal

import (
	"github.com/MrSnakeDoc/keg/internal/list"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured packages and their status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			onlyDeps, err := cmd.Flags().GetBool("deps")
			if err != nil {
				return err
			}

			l := list.New(cfg, nil)
			return l.Execute(cmd.Context(), onlyDeps)
		},
	}

	cmd.Flags().BoolP("deps", "d", false, "Show only non-config packages (deps/utils)")
	return cmd
}
