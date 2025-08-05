package internal

import (
	"github.com/MrSnakeDoc/keg/internal/add"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"

	"github.com/spf13/cobra"
)

func NewAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [packages...]",
		Short: "Add packages to configuration",
		Long: `Add packages to keg.yml configuration.
You can specify binary name if it differs from package name and mark packages as optional.

Examples:
  keg add bat                     # Add single package
  keg add bat starship            # Add multiple packages (all non-optional)
  keg add --optional bat starship # Add multiple optional packages
  keg add --binary=batcat bat     # Add package with different binary name (single package only)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			binary, err := cmd.Flags().GetString("binary")
			if err != nil {
				return err
			}

			optional, err := cmd.Flags().GetBool("optional")
			if err != nil {
				return err
			}

			// Create adder
			a := add.New(cfg, nil)

			return a.Execute(args, binary, optional)
		},
	}

	// Add flags
	cmd.Flags().StringP("binary", "b", "", "If binary name is different from package name")
	cmd.Flags().BoolP("optional", "o", false, "If package is optional")

	return cmd
}
