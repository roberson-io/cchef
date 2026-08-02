package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/filesig"
)

func init() {
	core.Register(ScanForEmbeddedFiles{})
}

// ScanForEmbeddedFiles looks for file signatures at every offset of the input.
// Ported from CyberChef ScanForEmbeddedFiles.mjs.
type ScanForEmbeddedFiles struct{}

// Meta returns the operation metadata.
func (ScanForEmbeddedFiles) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Scan for Embedded Files",
		Module:      "Default",
		Description: "Scans the data for potential embedded files by looking for magic bytes at all offsets. This operation is prone to false positives.\n\nWARNING: Files over about 100KB in size will take a VERY long time to process.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_file_signatures",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns one boolean per signature category, everything on except
// Miscellaneous, whose false positives drown the rest.
func (ScanForEmbeddedFiles) Args() []core.ArgDef {
	args := make([]core.ArgDef, len(detectFileTypeCats))
	for i, cat := range detectFileTypeCats {
		args[i] = core.ArgDef{Name: cat, Type: core.ArgBoolean, Value: cat != "Miscellaneous"}
	}
	return args
}

// Run reports every signature match in the input.
func (ScanForEmbeddedFiles) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var out strings.Builder
	out.WriteString("Scanning data for 'magic bytes' which may indicate embedded files. " +
		"The following results may be false positives and should not be treated as reliable. " +
		"Any sufficiently long file is likely to contain these magic bytes coincidentally.\n")

	categories := []string{}
	for i, cat := range detectFileTypeCats {
		if args[i].(bool) {
			categories = append(categories, cat)
		}
	}

	found := filesig.Scan(in.Bytes(), categories)
	for _, f := range found {
		fmt.Fprintf(&out, "\nOffset %d (0x%02x):\n  File type:   %s\n  Extension:   %s\n  MIME type:   %s\n",
			f.Offset, f.Offset, f.Details.Name, f.Details.Extension, f.Details.MIME)
		if f.Details.Description != "" {
			fmt.Fprintf(&out, "  Description: %s\n", f.Details.Description)
		}
	}
	if len(found) == 0 {
		out.WriteString("\nNo embedded files were found.")
	}

	return core.NewDish([]byte(out.String()), core.TypeString), nil
}
