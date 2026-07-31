package ops

import (
	"runtime"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// A short input must not cause a large allocation: a length or count read off
// the wire has to be checked against what is actually there before anything is
// sized from it.
func TestNoAllocationBombs(t *testing.T) {
	bombs := []string{
		"\xdd\xff\xff\xff\xff", "\xdc\xff\xff", "\xdf\xff\xff\xff\xff", "\xde\xff\xff",
		"\x9f", "\x8f",
		"\x9a\xff\xff\xff\xff", "\x9b\xff\xff\xff\xff\xff\xff\xff\xff",
		"\xbf\xff\xff\xff\xff", "\x5a\xff\xff\xff\xff", "\x7b\xff\xff\xff\xff\xff\xff\xff\xff",
		"\x12\xff\xff\xff\xff\xff\xff\xff\xff\x7f",
		"\x30\x82\xff\xff", "\x30\x84\xff\xff\xff\xff",
		"\x1f\x8b\x08\x00\xff\xff\xff\xff",
		"PK\x03\x04\xff\xff\xff\xff\xff\xff\xff\xff",
		"\xff\xff\xff\xff\xff\xff\xff\xff",
	}
	for _, name := range fuzzDecoderNames() {
		op, ok := core.Default.Get(name)
		if !ok {
			continue
		}
		args, err := core.CoerceArgs(op.Args(), nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, in := range bombs {
			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			func() {
				defer func() { _ = recover() }()
				_, _ = op.Run(core.NewDish([]byte(in), op.Meta().InputType), args)
			}()
			runtime.ReadMemStats(&after)
			if mb := float64(after.TotalAlloc-before.TotalAlloc) / (1 << 20); mb > 64 {
				t.Errorf("%s allocated %.0f MB from %d bytes of input (%q)", name, mb, len(in), in)
			}
		}
	}
}
