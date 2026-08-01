package ops

import (
	"strconv"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Fletcher8Checksum{})
	core.Register(Fletcher16Checksum{})
	core.Register(Fletcher32Checksum{})
	core.Register(Fletcher64Checksum{})
}

// fletcherHex renders v as hex, zero-padded to at least length digits (the
// uint64 counterpart of utilsHex, avoiding a narrowing conversion).
func fletcherHex(v uint64, length int) string {
	s := strconv.FormatUint(v, 16)
	for len(s) < length {
		s = "0" + s
	}
	return s
}

// The Fletcher checksums (Fletcher, 1982) are position-dependent running sums
// modulo 2^n − 1, ported from CyberChef's Fletcher{8,16,32,64}Checksum. Each
// takes raw bytes and returns the checksum as hex.

// fletcherMeta builds the shared metadata for a Fletcher operation of the given
// bit width.
func fletcherMeta(name, bits string) core.OpMeta {
	return core.OpMeta{
		Name:        name,
		Module:      "Crypto",
		Description: "The Fletcher-" + bits + " Checksum is an algorithm for computing a position-dependent checksum.",
		InfoURL:     "https://wikipedia.org/wiki/Fletcher%27s_checksum",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Fletcher8Checksum computes the 8-bit Fletcher checksum (two 4-bit sums mod 15).
type Fletcher8Checksum struct{}

// Meta returns the operation metadata.
func (Fletcher8Checksum) Meta() core.OpMeta { return fletcherMeta("Fletcher-8 Checksum", "8") }

// Args returns the argument definitions.
func (Fletcher8Checksum) Args() []core.ArgDef { return nil }

// Run computes the checksum.
func (Fletcher8Checksum) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	var a, b uint64
	for _, by := range in.Bytes() {
		a = (a + uint64(by)) % 0xf
		b = (b + a) % 0xf
	}
	return core.NewDish([]byte(fletcherHex((b<<4)|a, 2)), core.TypeString), nil
}

// Fletcher16Checksum computes the 16-bit Fletcher checksum (two 8-bit sums mod 255).
type Fletcher16Checksum struct{}

// Meta returns the operation metadata.
func (Fletcher16Checksum) Meta() core.OpMeta { return fletcherMeta("Fletcher-16 Checksum", "16") }

// Args returns the argument definitions.
func (Fletcher16Checksum) Args() []core.ArgDef { return nil }

// Run computes the checksum.
func (Fletcher16Checksum) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	var a, b uint64
	for _, by := range in.Bytes() {
		a = (a + uint64(by)) % 0xff
		b = (b + a) % 0xff
	}
	return core.NewDish([]byte(fletcherHex((b<<8)|a, 4)), core.TypeString), nil
}

// Fletcher32Checksum computes the 32-bit Fletcher checksum over 16-bit
// little-endian words (two 16-bit sums mod 65535).
type Fletcher32Checksum struct{}

// Meta returns the operation metadata.
func (Fletcher32Checksum) Meta() core.OpMeta { return fletcherMeta("Fletcher-32 Checksum", "32") }

// Args returns the argument definitions.
func (Fletcher32Checksum) Args() []core.ArgDef { return nil }

// Run computes the checksum.
func (Fletcher32Checksum) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	data := in.Bytes()
	var a, b uint64
	i := 0
	for ; i+2 <= len(data); i += 2 {
		word := uint64(data[i]) | uint64(data[i+1])<<8
		a = (a + word) % 0xffff
		b = (b + a) % 0xffff
	}
	if len(data)%2 != 0 {
		a = (a + uint64(data[len(data)-1])) % 0xffff
		b = (b + a) % 0xffff
	}
	return core.NewDish([]byte(fletcherHex((b<<16)|a, 8)), core.TypeString), nil
}

// Fletcher64Checksum computes the 64-bit Fletcher checksum over 32-bit
// little-endian words (two 32-bit sums mod 2^32 − 1).
type Fletcher64Checksum struct{}

// Meta returns the operation metadata.
func (Fletcher64Checksum) Meta() core.OpMeta { return fletcherMeta("Fletcher-64 Checksum", "64") }

// Args returns the argument definitions.
func (Fletcher64Checksum) Args() []core.ArgDef { return nil }

// Run computes the checksum.
func (Fletcher64Checksum) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	data := in.Bytes()
	var a, b uint64
	i := 0
	for ; i+4 <= len(data); i += 4 {
		word := uint64(data[i]) | uint64(data[i+1])<<8 | uint64(data[i+2])<<16 | uint64(data[i+3])<<24
		a = (a + word) % 0xffffffff
		b = (b + a) % 0xffffffff
	}
	if rem := len(data) % 4; rem != 0 {
		var last uint64
		for j := range rem {
			last = (last << 8) | uint64(data[len(data)-1-j])
		}
		a = (a + last) % 0xffffffff
		b = (b + a) % 0xffffffff
	}
	return core.NewDish([]byte(fletcherHex(b, 8)+fletcherHex(a, 8)), core.TypeString), nil
}
