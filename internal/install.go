package internal

import (
	"github.com/MrSnakeDoc/keg/internal/errs"
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

			addFlag, err := cmd.Flags().GetBool("add")
			if err != nil {
				return err
			}
			optFlag, err := cmd.Flags().GetBool("optional")
			if err != nil {
				return err
			}
			binaryFlag, err := cmd.Flags().GetString("binary")
			if err != nil {
				return err
			}

			err = validateFlags(allFlag, addFlag, optFlag, binaryFlag, args)
			if err != nil {
				return err
			}

			// Create a new installer instance
			inst := install.New(cfg, nil)

			return inst.Execute(args, allFlag, addFlag, optFlag, binaryFlag)
		},
	}

	// Add flags
	cmd.Flags().BoolP("all", "a", false, "Install all packages, including optionals")
	cmd.Flags().BoolP("add", "A", false, "Add specified package to the configuration if not present and install it")
	cmd.Flags().BoolP("optional", "o", false, "Mark added package as optional in the configuration (requires --add)")
	cmd.Flags().StringP("binary", "b", "", "Specify the binary name if it differs from the package name (requires --add)")

	return cmd
}

func validateFlags(all, add, opt bool, binary string, args []string) error {
	// Validate flag combinations
	if !all && len(args) == 0 {
		return middleware.FlagComboError(errs.ProvidePkgsOrAll, "Install", "install")
	}
	if all && len(args) > 0 {
		return middleware.FlagComboError(errs.AllWithNamedPackages, "Install", "install", "")
	}
	if all && add {
		return middleware.FlagComboError(errs.AllWithAddInvalid)
	}
	if (opt || binary != "") && !add {
		return middleware.FlagComboError(errs.OptOrBinRequireAdd)
	}
	if binary != "" && len(args) > 1 {
		return middleware.FlagComboError(errs.BinarySinglePackageOnly)
	}
	return nil
}
