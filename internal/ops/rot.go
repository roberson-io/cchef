package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ROT13{})
	core.Register(ROT47{})
	core.Register(ROT13BruteForce{})
	core.Register(ROT47BruteForce{})
}

// rot13Byte rotates a single byte by the ROT13 rule: letters by amount (mod 26)
// and digits by amountNum (mod 10), each gated by its rotate flag. Shared by ROT13
// and ROT13 Brute Force.
func rot13Byte(b byte, amount, amountNum int, rotLower, rotUpper, rotNum bool) byte {
	switch {
	case rotUpper && b >= 'A' && b <= 'Z':
		return byte('A' + (int(b)-'A'+amount)%26)
	case rotLower && b >= 'a' && b <= 'z':
		return byte('a' + (int(b)-'a'+amount)%26)
	case rotNum && b >= '0' && b <= '9':
		return byte('0' + (int(b)-'0'+amountNum)%10)
	default:
		return b
	}
}

// rot47Byte rotates a single printable-ASCII byte (33–126) by amount (mod 94).
// Shared by ROT47 and ROT47 Brute Force.
func rot47Byte(b byte, amount int) byte {
	if b >= 33 && b <= 126 {
		return byte(33 + (int(b)-33+amount)%94) // #nosec G115 -- result is always in [33,126]
	}
	return b
}

// ROT13 is a Caesar substitution cipher rotating alphabet characters (and
// optionally digits) by a configurable amount.
type ROT13 struct{}

// Meta returns the operation metadata.
func (ROT13) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT13",
		Module:      "Default",
		Description: "A simple Caesar substitution cipher which rotates alphabet characters by the specified amount (default 13).",
		InfoURL:     "https://wikipedia.org/wiki/ROT13",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ROT13) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Rotate lower case chars", Type: core.ArgBoolean, Value: true},
		{Name: "Rotate upper case chars", Type: core.ArgBoolean, Value: true},
		{Name: "Rotate numbers", Type: core.ArgBoolean, Value: false},
		{Name: "Amount", Type: core.ArgNumber, Value: 13},
	}
}

// Run applies the rotation. Ported from CyberChef ROT13.mjs.
func (ROT13) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rotLower := args[0].(bool)
	rotUpper := args[1].(bool)
	rotNumbers := args[2].(bool)
	amount := int(args[3].(float64))
	amountNumbers := amount

	out := append([]byte(nil), in.Bytes()...)
	if amount == 0 {
		return core.NewDish(out, core.TypeByteArray), nil
	}
	if amount < 0 {
		amount = 26 - (abs(amount) % 26)
		amountNumbers = 10 - (abs(amountNumbers) % 10)
	}

	for i, chr := range out {
		out[i] = rot13Byte(chr, amount, amountNumbers, rotLower, rotUpper, rotNumbers)
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// ROT47 rotates printable ASCII characters from 33 ('!') to 126 ('~').
type ROT47 struct{}

// Meta returns the operation metadata.
func (ROT47) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT47",
		Module:      "Default",
		Description: "A variant of the Caesar cipher covering ASCII characters 33 ('!') to 126 ('~'). Default rotation: 47.",
		InfoURL:     "https://wikipedia.org/wiki/ROT13#Variants",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (ROT47) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Amount", Type: core.ArgNumber, Value: 47},
	}
}

// Run applies the rotation. Ported from CyberChef ROT47.mjs.
func (ROT47) Run(in *core.Dish, args []any) (*core.Dish, error) {
	amount := int(args[0].(float64))
	out := append([]byte(nil), in.Bytes()...)
	if amount == 0 {
		return core.NewDish(out, core.TypeByteArray), nil
	}
	if amount < 0 {
		amount = 94 - (abs(amount) % 94)
	}
	for i, chr := range out {
		out[i] = rot47Byte(chr, amount)
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// rotBruteForce tries amounts 1..maxAmount-1, rotating each sample byte with rot,
// keeping results whose (lower-cased) text contains crib, and formatting each with
// an optional "Amount = NN: " prefix and whitespace escaping.
func rotBruteForce(sample []byte, maxAmount int, printAmount bool, crib string, rot func(b byte, amount int) byte) string {
	var output []string
	for amount := 1; amount < maxAmount; amount++ {
		rotated := make([]byte, len(sample))
		for i, b := range sample {
			rotated[i] = rot(b, amount)
		}
		s := byteArrayToUtf8(rotated)
		if crib != "" && !strings.Contains(strings.ToLower(s), crib) {
			continue
		}
		record := escapeWhitespace(s)
		if printAmount {
			record = fmt.Sprintf("Amount = %2d: ", amount) + record
		}
		output = append(output, record)
	}
	return strings.Join(output, "\n")
}

// ROT13BruteForce tries every ROT13 rotation, optionally filtering by a crib.
type ROT13BruteForce struct{}

// Meta returns the operation metadata.
func (ROT13BruteForce) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT13 Brute Force",
		Module:      "Default",
		Description: "Try all meaningful amounts for ROT13.<br><br>Optionally you can enter your known plaintext (crib) to filter the result.",
		InfoURL:     "https://wikipedia.org/wiki/ROT13",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ROT13BruteForce) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Rotate lower case chars", Type: core.ArgBoolean, Value: true},
		{Name: "Rotate upper case chars", Type: core.ArgBoolean, Value: true},
		{Name: "Rotate numbers", Type: core.ArgBoolean, Value: false},
		{Name: "Sample length", Type: core.ArgNumber, Value: float64(100)},
		{Name: "Sample offset", Type: core.ArgNumber, Value: float64(0)},
		{Name: "Print amount", Type: core.ArgBoolean, Value: true},
		{Name: "Crib (known plaintext string)", Type: core.ArgString, Value: ""},
	}
}

// Run enumerates the rotations. Ported from CyberChef ROT13BruteForce.mjs.
func (ROT13BruteForce) Run(in *core.Dish, args []any) (*core.Dish, error) {
	rotateLower, rotateUpper, rotateNum := args[0].(bool), args[1].(bool), args[2].(bool)
	sampleLength, sampleOffset := int(args[3].(float64)), int(args[4].(float64))
	printAmount := args[5].(bool)
	crib := strings.ToLower(args[6].(string))
	sample := sampleSlice(in.Bytes(), sampleOffset, sampleLength)

	rot := func(b byte, amount int) byte {
		return rot13Byte(b, amount, amount, rotateLower, rotateUpper, rotateNum)
	}
	out := rotBruteForce(sample, 26, printAmount, crib, rot)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// ROT47BruteForce tries every ROT47 rotation, optionally filtering by a crib.
type ROT47BruteForce struct{}

// Meta returns the operation metadata.
func (ROT47BruteForce) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "ROT47 Brute Force",
		Module:      "Default",
		Description: "Try all meaningful amounts for ROT47.<br><br>Optionally you can enter your known plaintext (crib) to filter the result.",
		InfoURL:     "https://wikipedia.org/wiki/ROT13#Variants",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ROT47BruteForce) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Sample length", Type: core.ArgNumber, Value: float64(100)},
		{Name: "Sample offset", Type: core.ArgNumber, Value: float64(0)},
		{Name: "Print amount", Type: core.ArgBoolean, Value: true},
		{Name: "Crib (known plaintext string)", Type: core.ArgString, Value: ""},
	}
}

// Run enumerates the rotations. Ported from CyberChef ROT47BruteForce.mjs.
func (ROT47BruteForce) Run(in *core.Dish, args []any) (*core.Dish, error) {
	sampleLength, sampleOffset := int(args[0].(float64)), int(args[1].(float64))
	printAmount := args[2].(bool)
	crib := strings.ToLower(args[3].(string))
	sample := sampleSlice(in.Bytes(), sampleOffset, sampleLength)

	out := rotBruteForce(sample, 94, printAmount, crib, rot47Byte)
	return core.NewDish([]byte(out), core.TypeString), nil
}
