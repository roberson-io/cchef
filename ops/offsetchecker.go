package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(OffsetChecker{})
}

// OffsetChecker highlights the offsets that are common across all samples.
type OffsetChecker struct{}

// Meta returns the operation metadata.
func (OffsetChecker) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Offset checker",
		Module:      "Default",
		Description: "Compares multiple samples and highlights (with <span> tags) the byte offsets that are identical across all of them.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (OffsetChecker) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Sample delimiter", Type: core.ArgString, Value: `\n\n`},
	}
}

// Run highlights common offsets. Ported from CyberChef OffsetChecker.mjs.
func (OffsetChecker) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sampleDelim := parseEscapedChars(args[0].(string))
	parts := strings.Split(in.String(), sampleDelim)
	if len(parts) < 2 {
		return nil, fmt.Errorf("not enough samples; modify the sample delimiter or add more data")
	}

	samples := make([][]rune, len(parts))
	for i, p := range parts {
		samples[i] = []rune(p)
	}
	outputs := make([]*strings.Builder, len(samples))
	for i := range outputs {
		outputs[i] = &strings.Builder{}
	}

	n := len(samples)
	s0 := samples[0]
	inMatch := false
	for i := range s0 {
		match := offsetAllMatch(samples, i, s0[i])
		for s := range n {
			inMatch = writeOffsetSample(outputs[s], samples, s0, s, i, n, match, inMatch)
		}
	}

	parts2 := make([]string, len(outputs))
	for i, o := range outputs {
		parts2[i] = o.String()
	}
	return core.NewDish([]byte(strings.Join(parts2, escapeHTML(sampleDelim))), core.TypeString), nil
}

// offsetAllMatch reports whether every other sample has s0's character chr at
// offset i (a sample too short to reach i does not match).
func offsetAllMatch(samples [][]rune, i int, chr rune) bool {
	for s := 1; s < len(samples); s++ {
		if i >= len(samples[s]) || samples[s][i] != chr {
			return false
		}
	}
	return true
}

// writeOffsetSample appends sample s's character at offset i to its output,
// opening/closing the highlight <span> as the match state changes. inMatch (the
// shared highlight state) is only updated on the last sample, so it is threaded
// through the return value. Ported faithfully from OffsetChecker.mjs.
func writeOffsetSample(out *strings.Builder, samples [][]rune, s0 []rune, s, i, n int, match, inMatch bool) bool {
	smp := samples[s]
	isLast := s == n-1
	if len(smp) <= i {
		if inMatch {
			out.WriteString("</span>")
		}
		if isLast {
			inMatch = false
		}
		return inMatch
	}
	cur := escapeHTML(string(smp[i]))
	switch {
	case match && !inMatch:
		out.WriteString("<span class='hl5'>" + cur)
		if len(smp) == i+1 {
			out.WriteString("</span>")
		}
		if isLast {
			inMatch = true
		}
	case !match && inMatch:
		out.WriteString("</span>" + cur)
		if isLast {
			inMatch = false
		}
	default:
		out.WriteString(cur)
		if inMatch && len(smp) == i+1 {
			out.WriteString("</span>")
			if len(smp)-1 != i {
				inMatch = false
			}
		}
	}
	if len(s0)-1 == i {
		writeOffsetTail(out, smp, i, inMatch)
	}
	return inMatch
}

// writeOffsetTail runs on the final offset of s0: it closes any open highlight
// and appends the remainder of a sample longer than s0.
func writeOffsetTail(out *strings.Builder, smp []rune, i int, inMatch bool) {
	if inMatch {
		out.WriteString("</span>")
	}
	out.WriteString(escapeHTML(string(smp[i+1:])))
}
