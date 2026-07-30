package yara

import (
	"fmt"
	"strconv"
	"strings"
)

// Rendering what a scan found, in the words CyberChef uses.

// Display says how much of what was found to write out.
type Display struct {
	Strings  bool
	Lengths  bool
	Meta     bool
	Counts   bool
	Warnings bool
	Console  bool
}

// Report writes what a scan found: the warnings, then anything the rules had to
// say, then the rules that held.
func Report(warnings []Warning, logs []string, results []Result, show Display) string {
	var b strings.Builder
	if show.Warnings {
		for _, w := range warnings {
			b.WriteString(w.String() + "\n")
		}
	}
	if show.Console {
		for _, line := range logs {
			b.WriteString(line + "\n")
		}
	}
	for _, r := range results {
		writeResult(&b, r, show)
	}
	return b.String()
}

// writeResult writes one rule that held.
func writeResult(b *strings.Builder, r Result, show Display) {
	meta := ""
	if show.Meta && len(r.Rule.Meta) > 0 {
		meta = " [" + metaText(r.Rule.Meta) + "]"
	}
	count := countText(len(r.Matches), show.Counts)

	// Where there is nothing to list, or nothing was asked for, the rule is
	// named on one line. The doubled space before the count is CyberChef's,
	// which puts a separator in front of a count that already carries one.
	if len(r.Matches) == 0 || (!show.Strings && !show.Lengths) {
		if count != "" {
			count = " " + count
		}
		fmt.Fprintf(b, "Input matches rule %q%s%s.\n", r.Rule.Name, meta, count)
		return
	}

	fmt.Fprintf(b, "Rule %q%s matches%s:\n", r.Rule.Name, meta, count)
	for _, m := range r.Matches {
		b.WriteString("Pos " + strconv.Itoa(m.Offset) + ", ")
		if show.Lengths {
			b.WriteString("length " + strconv.Itoa(m.Length) + ", ")
		}
		b.WriteString("identifier " + m.ID)
		if show.Strings {
			b.WriteString(`, data: "` + string(m.Data) + `"`)
		}
		b.WriteString("\n")
	}
}

// countText says how many times a rule's strings were found, when that was
// asked for. It carries its own leading space.
func countText(n int, wanted bool) string {
	if n == 0 || !wanted {
		return ""
	}
	if n == 1 {
		return " (1 time)"
	}
	return fmt.Sprintf(" (%d times)", n)
}

// metaText writes a rule's metadata as CyberChef does, keeping the order it was
// written in.
func metaText(meta []Meta) string {
	parts := make([]string, 0, len(meta))
	for _, m := range meta {
		parts = append(parts, m.Key+": "+metaValueText(m.Value))
	}
	return strings.Join(parts, ", ")
}

// metaValueText writes one metadata value plainly, whichever sort it is.
func metaValueText(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case int64:
		return strconv.FormatInt(value, 10)
	case bool:
		return strconv.FormatBool(value)
	}
	return ""
}
