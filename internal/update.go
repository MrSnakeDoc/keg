package internal

import (
	"context"

	"github.com/MrSnakeDoc/keg/internal/update"
	"github.com/spf13/cobra"
)

func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the keg CLI",
		Long: `Update the keg CLI to the latest version.
    
Examples:
  keg update            		# Update the keg CLI to the latest version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			update := update.New(nil, nil)
			return update.Execute(context.Background())
		},
	}
	return cmd
}
