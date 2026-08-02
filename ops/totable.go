package ops

import (
	"slices"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(ToTable{})
}

// parseCSV parses delimited data into rows of cells, honouring quoted fields.
// Ported from CyberChef Utils.parseCSV.
func parseCSV(data string, cellDelims, lineDelims []rune) [][]string {
	r := []rune(data)
	if len(r) > 0 && r[0] == '\uFEFF' { // strip BOM
		r = r[1:]
	}
	p := &csvParser{cellDelims: cellDelims, lineDelims: lineDelims}
	for i := 0; i < len(r); i++ {
		var next rune
		if i+1 < len(r) {
			next = r[i+1]
		}
		if p.feed(r[i], next) {
			i++ // a two-rune line delimiter (e.g. \r\n) was consumed
		}
	}
	return p.finish()
}

// csvParser is the state machine that splits delimited text into rows of cells,
// honouring quoted fields and "" escaped quotes.
type csvParser struct {
	cellDelims, lineDelims []rune
	lines                  [][]string
	line                   []string
	cell                   strings.Builder
	renderNext, inString   bool
}

// feed processes one rune (next is the following rune, for delimiter lookahead).
// It returns true when the following rune was also consumed as part of a
// two-rune line delimiter.
func (p *csvParser) feed(b, next rune) bool {
	switch {
	case p.renderNext:
		p.cell.WriteRune(b)
		p.renderNext = false
	case b == '"' && !p.inString:
		p.inString = true
	case b == '"' && p.inString:
		if next == '"' {
			p.renderNext = true
		} else {
			p.inString = false
		}
	case !p.inString && slices.Contains(p.cellDelims, b):
		p.endCell()
	case !p.inString && slices.Contains(p.lineDelims, b):
		p.endCell()
		p.lines = append(p.lines, p.line)
		p.line = nil
		return slices.Contains(p.lineDelims, next) && next != b
	default:
		p.cell.WriteRune(b)
	}
	return false
}

// endCell finishes the current cell and appends it to the current row.
func (p *csvParser) endCell() {
	p.line = append(p.line, p.cell.String())
	p.cell.Reset()
}

// finish flushes a trailing row (one without a final line delimiter) and returns
// all parsed rows.
func (p *csvParser) finish() [][]string {
	if len(p.line) > 0 {
		p.endCell()
		p.lines = append(p.lines, p.line)
	}
	return p.lines
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
	cellDelims := []rune(opsutil.ParseEscapedChars(args[0].(string)))
	rowDelims := []rune(opsutil.ParseEscapedChars(args[1].(string)))
	header := args[2].(bool)
	format := args[3].(string)

	data := parseCSV(opsutil.EscapeHTML(in.String()), cellDelims, rowDelims)
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
