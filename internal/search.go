package internal

import (
	"fmt"

	"github.com/MrSnakeDoc/keg/internal/middleware"
	"github.com/MrSnakeDoc/keg/internal/models"
	"github.com/MrSnakeDoc/keg/internal/search"

	"github.com/spf13/cobra"
)

func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [package]",
		Short: "Search for packages",
		Long: `Searches the locally cached index of Homebrew/core formulas
		and taps for packages matching the specified query. If no query is provided, it lists all available packages.
		You can use various flags to refine your search, such as exact matches, excluding descriptions, using regex, and more.
		For example:
		keg search <query> --exact --no-desc --json
		keg search <query> --regex --limit 20
		keg search --refresh
		keg search --json --limit 50
		keg search --exact --no-desc --regex --json --limit 10
		keg search --fzf/-f | fzf --with-nth=1,2,3,4 --delimiter="\t" --preview 'echo {} | awk -F"\t" "{print \$1}" | xargs brew info | bat -l md --style=plain --paging=never --color=always'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := middleware.Get[*models.Config](cmd, middleware.CtxKeyConfig)
			if err != nil {
				return err
			}

			exact, err := cmd.Flags().GetBool("exact")
			if err != nil {
				return err
			}

			noDesc, err := cmd.Flags().GetBool("no-desc")
			if err != nil {
				return err
			}

			regex, err := cmd.Flags().GetBool("regex")
			if err != nil {
				return err
			}

			fzf, err := cmd.Flags().GetBool("fzf")
			if err != nil {
				return err
			}

			jsonOut, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetInt("limit")
			if err != nil {
				return err
			}

			refresh, err := cmd.Flags().GetBool("refresh")
			if err != nil {
				return err
			}

			if refresh && (exact || noDesc || regex || fzf || jsonOut || limit > 0) {
				return fmt.Errorf("cannot use --refresh with other flags")
			}

			// Initialize Searcher with default store and HTTP client
			return search.New(nil, nil).Execute(args, nil, cfg, exact, noDesc, regex, fzf, jsonOut, limit, refresh, false)
		},
	}

	cmd.Flags().BoolP("exact", "e", false, "Search for exact matches")
	cmd.Flags().BoolP("no-desc", "d", false, "Exclude descriptions from search")
	cmd.Flags().BoolP("regex", "r", false, "Use regular expressions for search")
	cmd.Flags().BoolP("fzf", "f", false, "Output results in fzf-compatible format (name, aliases, desc separated by tabs)")
	cmd.Flags().Bool("json", false, "Output results in JSON format")
	cmd.Flags().IntP("limit", "l", 0, "Limit the number of results (0 for no limit)")
	cmd.Flags().BoolP("refresh", "R", false, "Force refresh of the package index")

	return cmd
}
