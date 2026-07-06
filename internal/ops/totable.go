package ops

import (
	"slices"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToTable{})
}

// htmlEscapes is CyberChef's Utils.escapeHtml replacement map.
var htmlEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
	"`", "&#x60;",
	"\x00", "",
)

// escapeHTML escapes HTML-significant characters (Utils.escapeHtml).
func escapeHTML(s string) string { return htmlEscaper.Replace(s) }

// parseCSV parses delimited data into rows of cells, honouring quoted fields.
// Ported from CyberChef Utils.parseCSV.
func parseCSV(data string, cellDelims, lineDelims []rune) [][]string {
	r := []rune(data)
	if len(r) > 0 && r[0] == '\uFEFF' { // strip BOM
		r = r[1:]
	}
	var lines [][]string
	var line []string
	var cell strings.Builder
	renderNext, inString := false, false

	for i := 0; i < len(r); i++ {
		b := r[i]
		var next rune
		if i+1 < len(r) {
			next = r[i+1]
		}
		switch {
		case renderNext:
			cell.WriteRune(b)
			renderNext = false
		case b == '"' && !inString:
			inString = true
		case b == '"' && inString:
			if next == '"' {
				renderNext = true
			} else {
				inString = false
			}
		case !inString && slices.Contains(cellDelims, b):
			line = append(line, cell.String())
			cell.Reset()
		case !inString && slices.Contains(lineDelims, b):
			line = append(line, cell.String())
			cell.Reset()
			lines = append(lines, line)
			line = nil
			if slices.Contains(lineDelims, next) && next != b {
				i++
			}
		default:
			cell.WriteRune(b)
		}
	}
	if len(line) > 0 {
		line = append(line, cell.String())
		lines = append(lines, line)
	}
	return lines
}

// ToTable renders delimited data as an ASCII, HTML or Markdown table.
type ToTable struct{}

// Meta returns the operation metadata.
func (ToTable) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Table",
		Module:      "Default",
		Description: "Renders delimited data (e.g. CSV) as an ASCII, HTML or Markdown table, with an optional header row.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToTable) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Cell delimiters", Type: core.ArgString, Value: ","},
		{Name: "Row delimiters", Type: core.ArgString, Value: `\r\n`},
		{Name: "Make first row header", Type: core.ArgBoolean, Value: false},
		{Name: "Format", Type: core.ArgOption, Value: []string{"ASCII", "HTML", "Markdown"}},
	}
}

// Run renders the table. Ported from CyberChef ToTable.mjs.
func (ToTable) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cellDelims := []rune(parseEscapedChars(args[0].(string)))
	rowDelims := []rune(parseEscapedChars(args[1].(string)))
	header := args[2].(bool)
	format := args[3].(string)

	data := parseCSV(escapeHTML(in.String()), cellDelims, rowDelims)
	if len(data) == 0 {
		return core.NewDish(nil, core.TypeString), nil
	}

	var out string
	switch format {
	case "ASCII":
		out = asciiTable(data, header)
	case "Markdown":
		out = markdownTable(data)
	default: // HTML
		out = htmlTable(data, header)
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// colWidths returns the maximum cell length per column.
func colWidths(data [][]string) []int {
	var w []int
	for _, row := range data {
		for i, cell := range row {
			for len(w) <= i {
				w = append(w, 0)
			}
			if len(cell) > w[i] {
				w[i] = len(cell)
			}
		}
	}
	return w
}

func paddedRow(row []string, widths []int) string {
	var b strings.Builder
	b.WriteByte('|')
	for i, cell := range row {
		b.WriteString(" " + cell + strings.Repeat(" ", widths[i]-len(cell)) + " |")
	}
	b.WriteByte('\n')
	return b.String()
}

func horizontalBorder(widths []int) string {
	var b strings.Builder
	b.WriteByte('+')
	for _, w := range widths {
		b.WriteString(strings.Repeat("-", w+2) + "+")
	}
	b.WriteByte('\n')
	return b.String()
}

func asciiTable(data [][]string, header bool) string {
	widths := colWidths(data)
	var b strings.Builder
	b.WriteString(horizontalBorder(widths))
	if header && len(data) > 0 {
		b.WriteString(paddedRow(data[0], widths))
		b.WriteString(horizontalBorder(widths))
		data = data[1:]
	}
	for _, row := range data {
		b.WriteString(paddedRow(row, widths))
	}
	b.WriteString(horizontalBorder(widths))
	return b.String()
}

func markdownTable(data [][]string) string {
	widths := colWidths(data)
	var b strings.Builder
	b.WriteString(paddedRow(data[0], widths))
	b.WriteByte('|')
	for i := range data[0] {
		b.WriteString(" " + strings.Repeat("-", widths[i]) + " |")
	}
	b.WriteByte('\n')
	for _, row := range data[1:] {
		b.WriteString(paddedRow(row, widths))
	}
	return b.String()
}

func htmlTable(data [][]string, header bool) string {
	var b strings.Builder
	b.WriteString("<table class='table table-hover table-sm table-bordered table-nonfluid'>")
	htmlRow := func(row []string, cell string) {
		b.WriteString("<tr>")
		for _, c := range row {
			b.WriteString("<" + cell + ">" + c + "</" + cell + ">")
		}
		b.WriteString("</tr>")
	}
	if header && len(data) > 0 {
		b.WriteString("<thead class='thead-light'>")
		htmlRow(data[0], "th")
		b.WriteString("</thead>")
		data = data[1:]
	}
	b.WriteString("<tbody>")
	for _, row := range data {
		htmlRow(row, "td")
	}
	b.WriteString("</tbody></table>")
	return b.String()
}
