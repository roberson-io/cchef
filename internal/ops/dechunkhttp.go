package ops

import (
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(DechunkHTTPResponse{})
}

// jsSlice mimics JavaScript String.prototype.slice(start, end) for non-negative
// indices, clamping to the string bounds instead of panicking.
func jsSlice(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if start > len(s) {
		start = len(s)
	}
	if end > len(s) {
		end = len(s)
	}
	if end < start {
		end = start
	}
	return s[start:end]
}

// leadingHex parses the leading hexadecimal integer of s the way JavaScript's
// parseInt(s, 16) does: skip leading whitespace, read hex digits, stop at the
// first non-hex character. Returns false (NaN) if there is no hex digit.
func leadingHex(s string) (int, bool) {
	s = strings.TrimLeft(s, " \t\n\r\f\v")
	i := 0
	for i < len(s) && isHexByte(s[i]) {
		i++
	}
	if i == 0 {
		return 0, false
	}
	v, err := strconv.ParseInt(s[:i], 16, 64)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

func isHexByte(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// DechunkHTTPResponse reassembles a chunked transfer-encoding HTTP body.
type DechunkHTTPResponse struct{}

// Meta returns the operation metadata.
func (DechunkHTTPResponse) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Dechunk HTTP response",
		Module:      "Default",
		Description: "Parses an HTTP response transferred using Transfer-Encoding: Chunked.",
		InfoURL:     "https://wikipedia.org/wiki/Chunked_transfer_encoding",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (DechunkHTTPResponse) Args() []core.ArgDef { return nil }

// Run reassembles the body. Ported from CyberChef DechunkHTTPResponse.mjs.
func (DechunkHTTPResponse) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	var chunks strings.Builder

	chunkSizeEnd := strings.IndexByte(input, '\n') + 1
	lineEndings := "\n"
	if chunkSizeEnd-2 >= 0 && input[chunkSizeEnd-2] == '\r' {
		lineEndings = "\r\n"
	}
	lel := len(lineEndings)

	chunkSize, ok := leadingHex(jsSlice(input, 0, chunkSizeEnd))
	for ok {
		if chunkSize == 0 {
			break
		}
		chunks.WriteString(jsSlice(input, chunkSizeEnd, chunkSize+chunkSizeEnd))
		input = jsSlice(input, chunkSizeEnd+chunkSize+lel, len(input))
		chunkSizeEnd = strings.Index(input, lineEndings) + lel
		chunkSize, ok = leadingHex(jsSlice(input, 0, chunkSizeEnd))
	}
	return core.NewDish([]byte(chunks.String()), core.TypeString), nil
}
