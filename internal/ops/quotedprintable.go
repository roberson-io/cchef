package ops

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ToQuotedPrintable{})
	core.Register(FromQuotedPrintable{})
}

// ToQuotedPrintable encodes bytes as Quoted-Printable (RFC 2045).
type ToQuotedPrintable struct{}

// Meta returns the operation metadata.
func (ToQuotedPrintable) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "To Quoted Printable",
		Module:      "Default",
		Description: "Quoted-Printable, or QP encoding, is an encoding using printable ASCII characters (alphanumeric and the equals sign '=') to transmit 8-bit data over a 7-bit data path. It is defined as a MIME content transfer encoding for use in email. QP uses '=' as an escape character and limits line length to 76.",
		InfoURL:     "https://wikipedia.org/wiki/Quoted-printable",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ToQuotedPrintable) Args() []core.ArgDef { return nil }

// QP soft-break helpers. After mimeEncode the string is pure ASCII, so all the
// substring/regex work below is byte-safe. Ported from ToQuotedPrintable.mjs.
var (
	qpNewlineRE = regexp.MustCompile(`\r\n|\n|\r`)
	// Trailing whitespace before a CRLF or at end of input. (JS uses /[\t ]+$/gm,
	// whose multiline $ also matches before \r; Go's (?m)$ does not, so the \r is
	// matched explicitly. Newlines are already normalised to \r\n at this point.)
	qpTrailingWSRE    = regexp.MustCompile(`[\t ]+(?:\r|$)`)
	qpSpaceBreakRE    = regexp.MustCompile("[ \t.,!?][^ \t.,!?]*$")
	qpIncompleteHexRE = regexp.MustCompile(`(?i)=[\da-f]{0,2}$`)
	qpIncomplete1RE   = regexp.MustCompile(`(?i)=[\da-f]?$`)
	qpFullHexRE       = regexp.MustCompile(`(?i)^(?:=[\da-f]{2}){1,4}$`)
	qpHexEndRE        = regexp.MustCompile(`(?i)=[\da-f]{2}$`)
)

// qpKeep reports whether a byte is emitted literally by mimeEncode (i.e. it
// falls in one of the printable ranges QP leaves untouched).
func qpKeep(b byte) bool {
	switch {
	case b == 0x09 || b == 0x0a || b == 0x0d || b == 0x20 || b == 0x21:
		return true
	case b >= 0x23 && b <= 0x3c:
		return true
	case b == 0x3e:
		return true
	case b >= 0x40 && b <= 0x5e:
		return true
	case b >= 0x60 && b <= 0x7e:
		return true
	}
	return false
}

// Run encodes the input. Ported from ToQuotedPrintable.mjs.
func (ToQuotedPrintable) Run(in *core.Dish, args []any) (*core.Dish, error) {
	var mb strings.Builder
	for _, b := range in.Bytes() {
		if qpKeep(b) {
			mb.WriteByte(b)
		} else {
			fmt.Fprintf(&mb, "=%02X", b)
		}
	}

	s := qpNewlineRE.ReplaceAllString(mb.String(), "\r\n")
	s = qpTrailingWSRE.ReplaceAllStringFunc(s, func(m string) string {
		term := ""
		if strings.HasSuffix(m, "\r") {
			term, m = "\r", m[:len(m)-1]
		}
		return strings.NewReplacer(" ", "=20", "\t", "=09").Replace(m) + term
	})

	return core.NewDish([]byte(qpSoftBreaks(s)), core.TypeString), nil
}

// qpSoftBreaks inserts QP soft line breaks so no line exceeds 76 characters,
// without splitting =XX escapes or multi-byte UTF-8 sequences. Matches the
// behaviour of mimelib's _addQPSoftLinebreaks (which CyberChef uses), verified
// against it directly in TestQPSoftBreaksDirect.
const (
	qpLineLengthMax = 76                  // RFC 2045 maximum encoded line length
	qpLineMargin    = qpLineLengthMax / 3 // window at the line end to search for a nicer break
	qpEscapeLen     = 3                   // length of an "=XX" escape
	qpASCIIMax      = 128                 // code points below this are single-byte ASCII
	qpUTF8LeadMin   = 0xc0                // start of the UTF-8 lead-byte range
)

func qpSoftBreaks(s string) string {
	n := len(s)
	pos := 0
	var result strings.Builder

	for pos < n {
		end := min(pos+qpLineLengthMax, n)
		line := s[pos:end]

		if idx := strings.Index(line, "\r\n"); idx >= 0 {
			line = line[:idx+2]
			result.WriteString(line)
			pos += len(line)
			continue
		}
		if strings.HasSuffix(line, "\n") {
			result.WriteString(line)
			pos += len(line)
			continue
		}

		// A newline within the end-of-line window ends the line there.
		tail := line[max(0, len(line)-qpLineMargin):]
		if nl := strings.IndexByte(tail, '\n'); nl >= 0 {
			matchLen := len(tail) - nl
			line = line[:len(line)-(matchLen-1)]
			result.WriteString(line)
			pos += len(line)
			continue
		}

		line = qpTrimLine(line, pos, n)
		line, pos = qpAppendSoftBreak(line, pos, n)
		result.WriteString(line)
	}

	return result.String()
}

// qpTrimLine shortens a full-width line to a nicer break point: at a space/
// punctuation boundary, before a trailing bare CR, or before an incomplete
// escape sequence.
func qpTrimLine(line string, pos, n int) string {
	tail := line[max(0, len(line)-qpLineMargin):]
	switch {
	case len(line) > qpLineLengthMax-qpLineMargin && qpSpaceBreakRE.MatchString(tail):
		m := qpSpaceBreakRE.FindString(tail)
		return line[:len(line)-(len(m)-1)]
	case strings.HasSuffix(line, "\r"):
		return line[:len(line)-1]
	case qpIncompleteHexRE.MatchString(line):
		return qpTrimIncompleteHex(line, pos, n)
	}
	return line
}

// qpTrimIncompleteHex trims a trailing incomplete "=" escape, then backs off
// whole "=XX" escapes that are the tail of a multi-byte UTF-8 sequence so a soft
// break never splits a character.
func qpTrimIncompleteHex(line string, pos, n int) string {
	if m := qpIncomplete1RE.FindString(line); m != "" {
		line = line[:len(line)-len(m)]
	}
	for len(line) > qpEscapeLen && len(line) < n-pos && !qpFullHexRE.MatchString(line) {
		m := qpHexEndRE.FindString(line)
		if m == "" {
			break
		}
		code, _ := strconv.ParseUint(m[1:3], 16, 16)
		if code < qpASCIIMax {
			break
		}
		line = line[:len(line)-qpEscapeLen]
		if code >= qpUTF8LeadMin {
			break
		}
	}
	return line
}

// qpAppendSoftBreak appends a soft line break ("=\r\n") to a non-final line,
// trimming a final escape/char if the line is exactly the maximum width, and
// advances pos. The final line is returned unchanged.
func qpAppendSoftBreak(line string, pos, n int) (string, int) {
	if pos+len(line) >= n || strings.HasSuffix(line, "\n") {
		return line, pos + len(line)
	}
	if len(line) == qpLineLengthMax && qpHexEndRE.MatchString(line) {
		line = line[:len(line)-qpEscapeLen]
	} else if len(line) == qpLineLengthMax {
		line = line[:len(line)-1]
	}
	pos += len(line)
	return line + "=\r\n", pos
}

// FromQuotedPrintable decodes a Quoted-Printable string back into raw bytes.
type FromQuotedPrintable struct{}

// Meta returns the operation metadata.
func (FromQuotedPrintable) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "From Quoted Printable",
		Module:      "Default",
		Description: "Converts QP-encoded text back into its raw byte value.",
		InfoURL:     "https://wikipedia.org/wiki/Quoted-printable",
		InputType:   core.TypeString,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (FromQuotedPrintable) Args() []core.ArgDef { return nil }

// qpSoftBreakRE matches a soft line break ("=" before CRLF/LF or end of input),
// which From QP strips before decoding.
var qpSoftBreakRE = regexp.MustCompile(`=(?:\r?\n|$)`)

// Run decodes the input. Ported from FromQuotedPrintable.mjs.
func (FromQuotedPrintable) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := qpSoftBreakRE.ReplaceAllString(in.String(), "")

	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '=' && i+2 < len(s) && isHexByte(s[i+1]) && isHexByte(s[i+2]) {
			v, _ := strconv.ParseUint(s[i+1:i+3], 16, 8)
			out = append(out, byte(v))
			i += 2
			continue
		}
		out = append(out, s[i])
	}
	return core.NewDish(out, core.TypeByteArray), nil
}
