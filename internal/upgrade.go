package internal

import (
	"github.com/MrSnakeDoc/keg/internal/upgrade"
	"github.com/MrSnakeDoc/keg/internal/utils"

	"github.com/spf13/cobra"
)

func NewUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [packages...]",
		Short: "Upgrade all installed packages",
		Long: `Upgrade Homebrew packages.
    
Examples:
  keg upgrade            		# Upgrades all packages from config
  keg upgrade bat fzf    		# Upgrades specific packages
  keg upgrade --check/-c 		# Checks for available upgrades
  keg upgrade --check/-c bat 	# Checks upgrades for specific package`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := utils.PreliminaryChecks()
			if err != nil {
				return err
			}

			// Check if we're just checking for updates
			checkOnly, err := cmd.Flags().GetBool("check")
			if err != nil {
				return err
			}

			upgrade := upgrade.New(cfg, nil)
			return upgrade.Execute(args, checkOnly)
		},
	}

	// Add flags
	cmd.Flags().BoolP("check", "c", false, "Check for available updates without installing them")

	return cmd
}
