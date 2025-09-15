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
		Long: `List all configured packages and their status.

By default shows only packages from your config.
With --deps/-d, shows extra packages that are installed but not configured.
With --fzf/-f, outputs in tab-separated format (package ↦ version ↦ status ↦ type),
ready to be piped into fzf or other tools.

Examples:
  # Show configured packages in a table
  keg list
  
  # Show only deps/utils (not in config)
  keg list --deps
  keg list -d
  
  # Output in fzf-friendly format (no table, tabs only)
  keg list --fzf
  keg list -f

  # Combine: list deps in fzf mode
  keg list -d -f

  # ⚡ Advanced: fuzzy-search packages interactively with fzf + bat
  keg list -f | fzf --with-nth=1,2,3,4 --delimiter="\t" --preview 'echo {} | awk -F"\t" "{print \$1}" | xargs brew info | bat -l md --style=plain --paging=never --color=always'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			onlyDeps, err := cmd.Flags().GetBool("deps")
			if err != nil {
				return err
			}

			return list.New(cfg, nil).Execute(cmd.Context(), onlyDeps)
		},
	}

	cmd.Flags().BoolP("deps", "d", false, "Show only non-config packages (deps/utils)")
	return cmd
}
