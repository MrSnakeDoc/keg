package internal

import (
	"github.com/spf13/cobra"
)

type CommandFactory func() *cobra.Command

var defaultCommands = []CommandFactory{
	NewBootstrapCmd,
	NewInitCmd,
	NewListCmd,
	NewDeployCmd,
	NewInstallCmd,
	NewUpgradeCmd,
	NewDeleteCmd,
	NewAddCmd,
	NewRemoveCmd,
	NewUpdateCmd,
}

func RegisterSubCommands(cmd *cobra.Command) {
	for _, factory := range defaultCommands {
		cmd.AddCommand(factory())
	}
}
