package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
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
		chr := s0[i]
		match := false
		for s := 1; s < n; s++ {
			if i >= len(samples[s]) || samples[s][i] != chr {
				match = false
				break
			}
			match = true
		}
		for s := range n {
			smp := samples[s]
			if len(smp) <= i {
				if inMatch {
					outputs[s].WriteString("</span>")
				}
				if s == n-1 {
					inMatch = false
				}
				continue
			}
			cur := escapeHTML(string(smp[i]))
			switch {
			case match && !inMatch:
				outputs[s].WriteString("<span class='hl5'>" + cur)
				if len(smp) == i+1 {
					outputs[s].WriteString("</span>")
				}
				if s == n-1 {
					inMatch = true
				}
			case !match && inMatch:
				outputs[s].WriteString("</span>" + cur)
				if s == n-1 {
					inMatch = false
				}
			default:
				outputs[s].WriteString(cur)
				if inMatch && len(smp) == i+1 {
					outputs[s].WriteString("</span>")
					if len(smp)-1 != i {
						inMatch = false
					}
				}
			}
			if len(s0)-1 == i {
				if inMatch {
					outputs[s].WriteString("</span>")
				}
				outputs[s].WriteString(escapeHTML(string(smp[i+1:])))
			}
		}
	}

	parts2 := make([]string, len(outputs))
	for i, o := range outputs {
		parts2[i] = o.String()
	}
	return core.NewDish([]byte(strings.Join(parts2, escapeHTML(sampleDelim))), core.TypeString), nil
}
