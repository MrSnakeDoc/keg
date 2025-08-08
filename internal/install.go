package internal

import (
	"github.com/MrSnakeDoc/keg/internal/install"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"

	"github.com/spf13/cobra"
)

func NewInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [packages...]",
		Short: "Installs the configured packages",
		Long: `Installs the packages defined in the configuration. By default, only installs non-optional packages.
    To install specific optional packages, list them as arguments.
    
Examples:
    keg install              # Installs only non-optional packages
    keg install lazygit asdf # Installs base packages + lazygit and asdf
    keg install --all        # Installs all packages, including optional ones`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			allFlag, err := cmd.Flags().GetBool("all")
			if err != nil {
				return err
			}
			interactive, err := cmd.Flags().GetBool("interactive")
			if err != nil {
				return err
			}

			// Create a new installer instance
			inst := install.New(cfg, nil, nil)

			return inst.Execute(args, allFlag, interactive)
		},
	}

	// Add flags
	cmd.Flags().BoolP("all", "a", false, "Install all packages, including optionals")
	cmd.Flags().BoolP("interactive", "i", false, "Prompt to add missing packages")

	return cmd
}
