package internal

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/logger"
	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/printer"
	"github.com/MrSnakeDoc/keg/internal/runner"
	"github.com/MrSnakeDoc/keg/internal/utils"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	r := &runner.ExecRunner{}
	return &cobra.Command{
		Use:   "list",
		Short: "List all configured packages and their status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			installed, err := utils.MapInstalledPackagesWith(r, func(pkg string) (string, bool) {
				return pkg, true
			})
			if err != nil {
				return fmt.Errorf("an error occurred while fetching installed packages: %w", err)
			}

			// Setup color functions
			p := printer.NewColorPrinter()

			// Create and configure table
			table := logger.CreateTable([]string{"Package", "Status", "Type"})

			// List all packages and their status
			for _, pkg := range cfg.Packages {
				name := pkg.Command
				if pkg.Binary != "" {
					name = pkg.Binary
				}

				status := p.Success("✓ installed")
				if !installed[name] {
					status = p.Error("✗ missing")
				}

				pkgType := "default"
				if pkg.Optional {
					pkgType = p.Warning("optional")
				}

				err = table.Append([]string{name, status, pkgType})
				if err != nil {
					return fmt.Errorf("an error occurred while appending to the table: %w", err)
				}
			}

			err = table.Render()
			if err != nil {
				return fmt.Errorf("an error occurred while rendering the table: %w", err)
			}

			return nil
		},
	}
}
