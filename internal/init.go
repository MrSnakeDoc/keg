package internal

import (
	"github.com/MrSnakeDoc/keg/internal/initiator"
	"github.com/MrSnakeDoc/keg/internal/logger"

	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize keg configuration in current directory",
		Long: `Initialize keg configuration.
This command will:
- Create keg.yml in the current directory
- Create the configuration directory in ~/.config/keg
- Save the packages file path in the global configuration`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := initiator.New().Execute(); err != nil {
				return err
			}

			logger.Success("Initialized keg in current directory")
			return nil
		},
	}
}
