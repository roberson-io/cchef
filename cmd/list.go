package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available operations grouped by category",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Group operations by category. Operations in more than one category
			// (e.g. URL Decode) appear under each.
			byCategory := map[string][]core.Operation{}
			for _, op := range core.Default.All() {
				for _, c := range categoriesOf(op.Meta().Name) {
					byCategory[c] = append(byCategory[c], op)
				}
			}

			categories := make([]string, 0, len(byCategory))
			for c := range byCategory {
				categories = append(categories, c)
			}
			sort.Strings(categories)

			var sb strings.Builder
			for _, c := range categories {
				fmt.Fprintf(&sb, "\n%s\n", c)
				w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
				for _, op := range byCategory[c] {
					meta := op.Meta()
					if _, err := fmt.Fprintf(w, "  %s\t%s\n", core.Kebab(meta.Name), summaryOf(meta)); err != nil {
						return err
					}
				}
				if err := w.Flush(); err != nil {
					return err
				}
			}
			_, err := io.WriteString(cmd.OutOrStdout(), sb.String())
			return err
		},
	}
	rootCmd.AddCommand(listCmd)
}
