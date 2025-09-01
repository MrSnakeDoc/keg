package internal

import (
	"github.com/MrSnakeDoc/keg/internal/errs"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/uninstall"

	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [packages...]",
		Short: "Delete installed packages",
		Long: `Delete packages installed via Homebrew.
You can delete specific packages or use --all to delete all packages from config.

Examples:
  keg delete bat             # Delete single package
  keg delete bat starship    # Delete multiple packages
  keg delete --all          # Delete all packages from config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			allFlag, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}

			removeFlag, err := cmd.Flags().GetBool("remove")
			if err != nil {
				return err
			}

			forceFlag, err := cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			// Validate flags combo
			if !allFlag && len(args) == 0 {
				return middleware.FlagComboError(errs.ProvidePkgsOrAll, "Delete", "delete")
			}
			if allFlag && len(args) > 0 {
				return middleware.FlagComboError(errs.AllWithNamedPackages, "Delete", "delete", "")
			}
			if removeFlag && allFlag && !forceFlag {
				return middleware.FlagComboError(errs.AllWithRemoveNeedsForce)
			}

			// Create uninstaller
			uninstall := uninstall.New(cfg, nil)

			return uninstall.Execute(args, allFlag, removeFlag, forceFlag)
		},
	}

	// Add flags
	cmd.Flags().BoolP("all", "a", false, "Delete all packages listed in keg.yml (system only)")
	cmd.Flags().BoolP("remove", "r", false, "Also remove package(s) from keg.yml after uninstall")
	cmd.Flags().BoolP("force", "f", false, "Required with --all --remove to purge the manifest")

	return cmd
}
