package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Lorenz{})
}

var (
	lorenzModels   = []string{"SZ40", "SZ42a", "SZ42b"}
	lorenzPatterns = []string{"KH Pattern", "ZMUG Pattern", "BREAM Pattern", "No Pattern", "Custom"}
	lorenzModes    = []string{"Send", "Receive"}
	lorenzTypes    = []string{"Plaintext", "ITA2"}
	lorenzFormats  = []string{"5/8/9", "+/-/."}
)

// Default lug (cam) patterns for each wheel, used as the Custom-pattern defaults.
const (
	lugPsi1 = ".x...xx.x.x..xxx.x.x.xxxx.x.x.x.x.x..x.xx.x"
	lugPsi2 = ".xx.x.xxx..x.x.x..x.xx.x.xxx.x....x.xx.x.x.x..x"
	lugPsi3 = ".x.x.x..xxx....x.x.xx.x.x.x..xxx.x.x..x.x.xx..x.x.x"
	lugPsi4 = ".xx...xxxxx.x.x.xx...x.xx.x.x..x.x.xx.x..x.x.x.x.x.x."
	lugPsi5 = "xx...xx.x..x.xx.x...x.x.x.x.x.x.x.x.xx..xxxx.x.x...xx.x..x."
	lugM37  = "x.x.x.x.x.x...x.x.x...x.x.x...x.x...."
	lugM61  = ".xxxx.xxxx.xxx.xxxx.xx....xxx.xxxx.xxxx.xxxx.xxxx.xxx.xxxx..."
	lugChi1 = ".x...xxx.x.xxxx.x...x.x..xxx....xx.xxxx.."
	lugChi2 = "x..xxx...x.xxxx..xx..x..xx.xx.."
	lugChi3 = "..xx..x.xxx...xx...xx..xx.xx."
	lugChi4 = "xx..x..xxxx..xx.xxx....x.."
	lugChi5 = "xx..xx....xxxx.x..x.x.."
)

// Lorenz emulates the Lorenz SZ40/42 cipher attachment.
type Lorenz struct{}

// Meta returns the operation metadata.
func (Lorenz) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Lorenz",
		Module:      "Bletchley",
		Description: "The Lorenz SZ40/42 cipher attachment was a WW2 German rotor cipher machine with twelve rotors which attached in-line between remote teleprinters. It used the Vernam cipher with two groups of five rotors (the psi and chi wheels) plus two motor wheels to generate a pseudorandom ITA2 key stream XORed with the plaintext.",
		InfoURL:     "https://wikipedia.org/wiki/Lorenz_cipher",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions (31 args, matching CyberChef's layout).
func (Lorenz) Args() []core.ArgDef {
	defs := []core.ArgDef{
		{Name: "Model", Type: core.ArgOption, Value: lorenzModels},
		{Name: "Wheel Pattern", Type: core.ArgOption, Value: lorenzPatterns},
		{Name: "KT-Schalter", Type: core.ArgBoolean, Value: false},
		{Name: "Mode", Type: core.ArgOption, Value: lorenzModes},
		{Name: "Input Type", Type: core.ArgOption, Value: lorenzTypes},
		{Name: "Output Type", Type: core.ArgOption, Value: lorenzTypes},
		{Name: "ITA2 Format", Type: core.ArgOption, Value: lorenzFormats},
	}
	starts := []string{
		"Ψ1 start (1-43)", "Ψ2 start (1-47)", "Ψ3 start (1-51)", "Ψ4 start (1-53)",
		"Ψ5 start (1-59)", "Μ37 start (1-37)", "Μ61 start (1-61)", "Χ1 start (1-41)",
		"Χ2 start (1-31)", "Χ3 start (1-29)", "Χ4 start (1-26)", "Χ5 start (1-23)",
	}
	for _, name := range starts {
		defs = append(defs, core.ArgDef{Name: name, Type: core.ArgNumber, Value: float64(1)})
	}
	lugs := []struct {
		name string
		def  string
	}{
		{"Ψ1 lugs (43)", lugPsi1},
		{"Ψ2 lugs (47)", lugPsi2},
		{"Ψ3 lugs (51)", lugPsi3},
		{"Ψ4 lugs (53)", lugPsi4},
		{"Ψ5 lugs (59)", lugPsi5},
		{"Μ37 lugs (37)", lugM37},
		{"Μ61 lugs (61)", lugM61},
		{"Χ1 lugs (41)", lugChi1},
		{"Χ2 lugs (31)", lugChi2},
		{"Χ3 lugs (29)", lugChi3},
		{"Χ4 lugs (26)", lugChi4},
		{"Χ5 lugs (23)", lugChi5},
	}
	for _, l := range lugs {
		defs = append(defs, core.ArgDef{Name: l.name, Type: core.ArgString, Value: l.def})
	}
	return defs
}

// Run enciphers/deciphers with the Lorenz machine. Ported from CyberChef Lorenz.mjs.
func (Lorenz) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cfg := lorenzConfig{
		model:   args[0].(string),
		pattern: args[1].(string),
		kt:      args[2].(bool),
		mode:    args[3].(string),
		intype:  args[4].(string),
		outtype: args[5].(string),
		format:  args[6].(string),
	}
	if err := lorenzValidateStarts(args); err != nil {
		return nil, err
	}
	settings, err := lorenzBuildSettings(cfg.pattern, args)
	if err != nil {
		return nil, err
	}

	ita2Input, err := lorenzToITA2(in.String(), cfg.intype, cfg.mode)
	if err != nil {
		return nil, err
	}

	m := newLorenzMachine(cfg, settings, args)
	ita2Output := m.encipherAll(ita2Input)

	out := lorenzFromITA2(ita2Output, cfg.outtype, cfg.mode)
	return core.NewDish([]byte(out), core.TypeString), nil
}

// lorenzConfig holds the non-wheel settings.
type lorenzConfig struct {
	model, pattern, mode, intype, outtype, format string
	kt                                            bool
}

// lorenzStartBound describes one rotor-start range check.
type lorenzStartBound struct {
	idx   int
	label string
	max   int
}

var lorenzStartBounds = []lorenzStartBound{
	{7, "Ψ1", 43},
	{8, "Ψ2", 47},
	{9, "Ψ3", 51},
	{10, "Ψ4", 53},
	{11, "Ψ5", 59},
	{12, "Μ37", 37},
	{13, "Μ61", 61},
	{14, "Χ1", 41},
	{15, "Χ2", 31},
	{16, "Χ3", 29},
	{17, "Χ4", 26},
	{18, "Χ5", 23},
}

// lorenzValidateStarts range-checks each rotor start position.
func lorenzValidateStarts(args []any) error {
	for _, b := range lorenzStartBounds {
		if v := colNum(args[b.idx]); v < 1 || v > b.max {
			return fmt.Errorf("%s start must be between 1 and %d", b.label, b.max) //nolint:staticcheck,revive // verbatim CyberChef message
		}
	}
	return nil
}

// lorenzCustomLug describes one custom-lug argument: its index, required length,
// and the (verbatim, Greek/Latin-inconsistent) error message on failure.
type lorenzCustomLug struct {
	idx    int
	length int
	errMsg string
}

var lorenzCustomLugs = []lorenzCustomLug{
	{19, 43, "Ψ1 custom lugs must be 43 long and can only include . or x "},
	{20, 47, "Ψ2 custom lugs must be 47 long and can only include . or x"},
	{21, 51, "Ψ3 custom lugs must be 51 long and can only include . or x"},
	{22, 53, "Ψ4 custom lugs must be 53 long and can only include . or x"},
	{23, 59, "Ψ5 custom lugs must be 59 long and can only include . or x"},
	{24, 37, "M37 custom lugs must be 37 long and can only include . or x"},
	{25, 61, "M61 custom lugs must be 61 long and can only include . or x"},
	{26, 41, "Χ1 custom lugs must be 41 long and can only include . or x"},
	{27, 31, "Χ2 custom lugs must be 31 long and can only include . or x"},
	{28, 29, "Χ3 custom lugs must be 29 long and can only include . or x"},
	{29, 26, "Χ4 custom lugs must be 26 long and can only include . or x"},
	{30, 23, "Χ5 custom lugs must be 23 long and can only include . or x"},
}

// lorenzBuildSettings returns the chosen wheel bit-patterns, either a named
// preset or a validated custom set read from the lug arguments.
func lorenzBuildSettings(pattern string, args []any) (lorenzRings, error) {
	if pattern != "Custom" {
		return initPatterns[pattern], nil
	}
	for _, cl := range lorenzCustomLugs {
		s := args[cl.idx].(string)
		if len(s) != cl.length || !isLugString(s) {
			return lorenzRings{}, fmt.Errorf("%s", cl.errMsg) //nolint:staticcheck,revive // verbatim CyberChef message
		}
	}
	// Build a fresh ring set from the lugs (never mutate the shared No Pattern).
	return lorenzRings{
		S: map[int][]int{
			1: readLugs(args[19].(string)), 2: readLugs(args[20].(string)),
			3: readLugs(args[21].(string)), 4: readLugs(args[22].(string)),
			5: readLugs(args[23].(string)),
		},
		M: map[int][]int{
			1: readLugs(args[25].(string)), // M61
			2: readLugs(args[24].(string)), // M37
		},
		X: map[int][]int{
			1: readLugs(args[26].(string)), 2: readLugs(args[27].(string)),
			3: readLugs(args[28].(string)), 4: readLugs(args[29].(string)),
			5: readLugs(args[30].(string)),
		},
	}, nil
}

// isLugString reports whether s consists solely of '.', 'x' or 'X' (regex ^[.xX]*$).
func isLugString(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '.' && s[i] != 'x' && s[i] != 'X' {
			return false
		}
	}
	return true
}

// readLugs converts a lug string to 0/1 bits ('.' → 0, anything else → 1).
func readLugs(s string) []int {
	out := make([]int, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '.' {
			out[i] = 1
		}
	}
	return out
}

// lorenzToITA2 converts the input to an ITA2 stream. In ITA2/Receive modes the
// input must already be valid ITA2; in Plaintext/Send mode it is converted with
// figure/letter shifts. Ported from Lorenz.mjs convertToITA2.
func lorenzToITA2(input, intype, mode string) (string, error) {
	if intype == "ITA2" || mode == "Receive" {
		return lorenzValidateITA2(input)
	}
	return lorenzPlaintextToITA2(input)
}

// lorenzValidateITA2 upper-cases and validates each character as ITA2.
func lorenzValidateITA2(input string) (string, error) {
	var b strings.Builder
	for _, r := range input {
		letter := strings.ToUpper(string(r))
		if !strings.Contains(validITA2, letter) {
			errltr := letter
			switch r {
			case '\n':
				errltr = "Carriage Return"
			case ' ':
				errltr = "Space"
			}
			return "", fmt.Errorf("Invalid ITA2 character : %s", errltr) //nolint:staticcheck,revive // verbatim CyberChef message
		}
		b.WriteString(letter)
	}
	return b.String(), nil
}

// lorenzPlaintextToITA2 converts plaintext to ITA2, inserting figure-shift (55),
// letter-shift (88), carriage-return (34) and line-feed (4) sequences.
func lorenzPlaintextToITA2(input string) (string, error) {
	var b strings.Builder
	figShifted := false
	for _, r := range input {
		letter := strings.ToUpper(string(r))
		if !strings.Contains(lorenzValidChars, letter) {
			return "", fmt.Errorf("Invalid Plaintext character : %s", letter) //nolint:staticcheck,revive // verbatim CyberChef message
		}
		figChar := strings.Contains(lorenzFigShiftedChars, letter)
		switch {
		case !figShifted && figChar:
			figShifted = true
			b.WriteString("55" + lorenzFigShift[letter])
		case figShifted:
			switch {
			case r == '\n':
				b.WriteString("34")
			case r == '\r':
				b.WriteString("4")
			case !figChar:
				figShifted = false
				b.WriteString("88" + letter)
			default:
				b.WriteString(lorenzFigShift[letter])
			}
		default:
			switch r {
			case '\n':
				b.WriteString("34")
			case '\r':
				b.WriteString("4")
			default:
				b.WriteString(letter)
			}
		}
	}
	return b.String(), nil
}

// lorenzFromITA2 converts the enciphered ITA2 output to the requested output
// type. Only Receive + Plaintext performs figure/letter-shift decoding.
func lorenzFromITA2(input, outtype, mode string) string {
	if mode != "Receive" || outtype != "Plaintext" {
		return input
	}
	var b strings.Builder
	figShifted := false
	for _, r := range input {
		letter := string(r)
		switch letter {
		case "5", "+":
			figShifted = true
		case "8", "-":
			figShifted = false
		case "9":
			b.WriteString(" ")
		case "3":
			b.WriteString("\n")
		case "4":
			// carriage return produces no output
		case "/":
			b.WriteString("/")
		default:
			if figShifted {
				b.WriteString(lorenzReverseFigShift[letter])
			} else {
				b.WriteString(letter)
			}
		}
	}
	return b.String()
}

// Lorenz-specific ITA2 resources (plaintext<->ITA2 figure/letter shifting),
// ported from CyberChef Lorenz.mjs.

const (
	lorenzValidChars      = "ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890+-'()/:=?,. \n\r"
	lorenzFigShiftedChars = "1234567890+-'()/:=?,."
)

// figShiftArr maps a plaintext character to its ITA2 figure-shift letter.
var lorenzFigShift = map[string]string{
	"1":  "Q",
	"2":  "W",
	"3":  "E",
	"4":  "R",
	"5":  "T",
	"6":  "Y",
	"7":  "U",
	"8":  "I",
	"9":  "O",
	"0":  "P",
	" ":  "9",
	"-":  "A",
	"?":  "B",
	":":  "C",
	"#":  "D",
	"%":  "F",
	"@":  "G",
	"£":  "H",
	"":   "J",
	"(":  "K",
	")":  "L",
	".":  "M",
	",":  "N",
	"'":  "S",
	"=":  "V",
	"/":  "X",
	"+":  "Z",
	"\n": "3",
	"\r": "4",
}

// reverseFigShift maps an ITA2 figure-shift letter back to its plaintext char.
var lorenzReverseFigShift = map[string]string{
	"Q": "1",
	"W": "2",
	"E": "3",
	"R": "4",
	"T": "5",
	"Y": "6",
	"U": "7",
	"I": "8",
	"O": "9",
	"P": "0",
	"9": " ",
	"A": "-",
	"B": "?",
	"C": ":",
	"D": "#",
	"F": "%",
	"G": "@",
	"H": "£",
	"J": "",
	"K": "(",
	"L": ")",
	"M": ".",
	"N": ",",
	"S": "'",
	"V": "=",
	"X": "/",
	"Z": "+",
	"3": "\n",
	"4": "\r",
}

// reverseITA2 maps a 5-bit ITA2 code to its character. Where several
// characters share a code, the last in ITA2_TABLE order wins (matching JS).
var lorenzReverseITA2 = map[string]string{
	"11000": "A",
	"10011": "B",
	"01110": "C",
	"10010": "D",
	"10000": "E",
	"10110": "F",
	"01011": "G",
	"00101": "H",
	"01100": "I",
	"11010": "J",
	"11110": "K",
	"01001": "L",
	"00111": "M",
	"00110": "N",
	"00011": "O",
	"01101": "P",
	"11101": "Q",
	"01010": "R",
	"10100": "S",
	"00001": "T",
	"11100": "U",
	"01111": "V",
	"11001": "W",
	"10111": "X",
	"10101": "Y",
	"10001": "Z",
	"00010": "3",
	"01000": "4",
	"00100": ".",
	"00000": "/",
	"11111": "-",
	"11011": "+",
}
