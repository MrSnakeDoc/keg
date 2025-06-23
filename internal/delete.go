package internal

import (
	"github.com/MrSnakeDoc/keg/internal/uninstall"
	"github.com/MrSnakeDoc/keg/internal/utils"

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
			cfg, err := utils.PreliminaryChecks()
			if err != nil {
				return err
			}

			allFlag, _ := cmd.Flags().GetBool("all")

			// Create uninstaller
			uninstall := uninstall.New(cfg, nil)

			return uninstall.Execute(args, allFlag)
		},
	}

	// Add flags
	cmd.Flags().BoolP("all", "a", false, "Delete all packages from config")
	cmd.Flags().BoolP("permanently", "p", false, "Delete packages permanently from config")

	return cmd
}
