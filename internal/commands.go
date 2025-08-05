package internal

import (
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/spf13/cobra"
)

var defaultCommands = []middleware.CommandFactory{
	NewInitCmd,
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewListCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled)(NewBootstrapCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewDeployCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewInstallCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewUpgradeCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewDeleteCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewAddCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewRemoveCmd),
	middleware.UseMiddlewareChain(middleware.RequireConfig, middleware.IsHomebrewInstalled, middleware.LoadPkgList)(NewUpdateCmd),
}

func RegisterSubCommands(cmd *cobra.Command) {
	for _, factory := range defaultCommands {
		cmd.AddCommand(factory())
	}
}
