package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

// listEntry is one operation in the machine-readable listing: what a completion
// script or a wrapper needs to know about a subcommand.
type listEntry struct {
	Command    string   `json:"command"`
	Name       string   `json:"name"`
	Summary    string   `json:"summary"`
	Categories []string `json:"categories"`
}

// listJSON writes every operation as JSON, sorted by subcommand so the output
// is stable between runs.
func listJSON(w io.Writer) error {
	ops := core.Default.All()
	entries := make([]listEntry, 0, len(ops))
	for _, op := range ops {
		meta := op.Meta()
		entries = append(entries, listEntry{
			Command:    core.Kebab(meta.Name),
			Name:       meta.Name,
			Summary:    summaryOf(meta),
			Categories: categoriesOf(meta.Name),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Command < entries[j].Command })
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

// flagListJSON selects the machine-readable listing.
var flagListJSON bool

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available operations grouped by category",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if flagListJSON {
				return listJSON(cmd.OutOrStdout())
			}
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
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "list operations as JSON, for scripts and completions")
	rootCmd.AddCommand(listCmd)
}
