package internal

import (
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/spf13/cobra"
)

var defaultCommands = []middleware.CommandFactory{
	NewInitCmd,
	NewListCmd,
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewBootstrapCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewDeployCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewInstallCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewUpgradeCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewDeleteCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewAddCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewRemoveCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig)(NewUpdateCmd),
}

func RegisterSubCommands(cmd *cobra.Command) {
	for _, factory := range defaultCommands {
		cmd.AddCommand(factory())
	}
}
