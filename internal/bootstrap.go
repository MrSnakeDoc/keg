package internal

import (
	"github.com/MrSnakeDoc/keg/internal/bootstraper"

	"github.com/spf13/cobra"
)

func NewBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Install and set zsh as default shell",
		Long: `Install and set zsh as default shell.
    This command will:
    - Check if zsh is installed
    - If not, prompt to install zsh
    - Set zsh as the default shell`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Create bootstraper
			boot := bootstraper.New(nil)

			// Run deployment
			return boot.Execute()
		},
	}
}
