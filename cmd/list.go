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
		Short: "List available operations grouped by module",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Group operation names by module.
			byModule := map[string][]core.Operation{}
			for _, op := range core.Default.All() {
				m := op.Meta().Module
				byModule[m] = append(byModule[m], op)
			}

			modules := make([]string, 0, len(byModule))
			for m := range byModule {
				modules = append(modules, m)
			}
			sort.Strings(modules)

			var sb strings.Builder
			for _, m := range modules {
				fmt.Fprintf(&sb, "\n%s\n", m)
				w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
				for _, op := range byModule[m] {
					meta := op.Meta()
					if _, err := fmt.Fprintf(w, "  %s\t%s\n", core.Kebab(meta.Name), meta.Name); err != nil {
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
