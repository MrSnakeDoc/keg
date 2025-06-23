package internal

import (
	"github.com/MrSnakeDoc/keg/internal/deploy"
	"github.com/MrSnakeDoc/keg/internal/utils"

	"github.com/spf13/cobra"
)

func NewDeployCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deploy",
		Short: "Deploy the complete development environment",
		Long: `Deploy and configure the complete development environment.
This includes:
- Installing ZSH and setting it as default shell
- Installing Homebrew if not present
- Installing all configured packages
- Running post-installation steps`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := utils.PreliminaryChecks()
			if err != nil {
				return err
			}

			// Create deployer
			dep := deploy.New(cfg, nil)

			// Run deployment
			return dep.Execute()
		},
	}
}
