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
			check, err := cmd.Flags().GetBool("check")
			if err != nil {
				return err
			}

			return update.New(nil, nil, nil).Execute(context.Background(), check)
		},
	}

	cmd.Flags().BoolP("check", "c", false, "Check for update without waiting for the next scheduled check")
	return cmd
}
