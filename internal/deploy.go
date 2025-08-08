package internal

import (
	"github.com/MrSnakeDoc/keg/internal/deploy"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
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
