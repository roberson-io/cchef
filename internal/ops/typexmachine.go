package ops

import (
	"errors"
	"strings"
)

// Typex machine, ported from CyberChef lib/Typex.mjs. It reuses the Enigma rotor
// and reflector types (enigma.go); Typex differs in having five rotors (the two
// right-hand ones static), reversible rotors, a Rotor-based input plugboard, and
// a special keyboard mode for symbols.

// typexRotors are the example rotor wirings (no real Typex wirings are public, so
// CyberChef ships a randomised set). Index i is the editableOption default for
// the (i+1)th rotor.
var typexRotors = []struct{ name, value string }{
	{"Example 1", "MCYLPQUVRXGSAOWNBJEZDTFKHI<BFHNQUW"},
	{"Example 2", "KHWENRCBISXJQGOFMAPVYZDLTU<BFHNQUW"},
	{"Example 3", "BYPDZMGIKQCUSATREHOJNLFWXV<BFHNQUW"},
	{"Example 4", "ZANJCGDLVHIXOBRPMSWQUKFYET<BFHNQUW"},
	{"Example 5", "QXBGUTOVFCZPJIHSWERYNDAMLK<BFHNQUW"},
	{"Example 6", "BDCNWUEIQVFTSXALOGZJYMHKPR<BFHNQUW"},
	{"Example 7", "WJUKEIABMSGFTQZVCNPHORDXYL<BFHNQUW"},
	{"Example 8", "TNVCZXDIPFWQKHSJMAOYLEURGB<BFHNQUW"},
}

// typexReflectors is the example reflector (also randomised).
var typexReflectors = []struct{ name, value string }{
	{"Example", "AN BC FG IE KD LU MH OR TS VZ WQ XJ YP"},
}

// typexKeyboardFwd maps a letter key to the symbol it produces on the Typex
// keyboard; typexKeyboardRev is the inverse. Note the £ sign is a non-ASCII rune.
var typexKeyboardFwd = map[rune]rune{
	'Q': '1', 'W': '2', 'E': '3', 'R': '4', 'T': '5', 'Y': '6', 'U': '7', 'I': '8', 'O': '9', 'P': '0',
	'A': '-', 'S': '/', 'D': 'Z', 'F': '%', 'G': 'X', 'H': '£', 'K': '(', 'L': ')',
	'C': 'V', 'B': '\'', 'N': ',', 'M': '.',
}

var typexKeyboardRev = func() map[rune]rune {
	m := make(map[rune]rune, len(typexKeyboardFwd))
	for k, v := range typexKeyboardFwd {
		m[v] = k
	}
	return m
}()

// newTypexRotor builds a Typex rotor. Unlike an Enigma rotor it can be reversed:
// the wiring is remapped so the rotor runs backwards, then handed to the shared
// Enigma rotor constructor (which validates and applies the ring setting).
func newTypexRotor(wiring, steps string, reversed bool, ringSetting, initialPos string) (*enigmaRotor, error) {
	wiringMod := wiring
	if reversed && enigmaWiringRe.MatchString(wiring) {
		var out [26]byte
		for i := range 26 {
			input := mod26(26-int(wiring[i]-'A'), 26)
			out[input] = byte('A' + mod26(26-i, 26)) // #nosec G115 -- mod26(...) is in [0,25], so 'A'+it is a letter
		}
		wiringMod = string(out[:])
	}
	return newEnigmaRotor(wiringMod, steps, ringSetting, initialPos)
}

// newTypexPlugboard builds the Typex input plugboard. Unlike Enigma's plugboard
// it allows an arbitrary map (entered like a rotor), and implements Typex's
// backwards input wiring by mirroring the alphabet before constructing a rotor.
func newTypexPlugboard(wiring string) (*enigmaRotor, error) {
	if !enigmaWiringRe.MatchString(wiring) {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Plugboard wiring must be 26 unique uppercase letters")
	}
	var mirrored [26]byte
	for i := range 26 {
		mirrored[i] = byte('A' + mod26(26-int(wiring[i]-'A'), 26)) // #nosec G115 -- mirrored index is a letter in [A,Z]
	}
	r, err := newEnigmaRotor(string(mirrored[:]), "", "A", "A")
	if err != nil {
		return nil, errors.New(strings.Replace(err.Error(), "Rotor", "Plugboard", 1))
	}
	return r, nil
}

// typexParseRotorStr splits a rotor spec into wiring and stepping parts. The
// empty-rotor message reproduces CyberChef's (which passes an undefined index).
func typexParseRotorStr(rotor string) (wiring, steps string, err error) {
	if rotor == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return "", "", errors.New("Rotor undefined must be provided.")
	}
	if !strings.Contains(rotor, "<") {
		return rotor, "", nil
	}
	parts := strings.Split(rotor, "<")
	return parts[0], parts[1], nil
}

// typexKeyboardEncode maps spaces and keyboard symbols to their letter shift
// sequences before enciphering (Typex keyboard emulation, Encrypt direction).
func typexKeyboardEncode(input string) string {
	var b strings.Builder
	mode := false // true = in symbol mode
	for _, x := range input {
		switch {
		case x == ' ':
			b.WriteByte('X')
		case mode:
			if r, ok := typexKeyboardRev[x]; ok {
				b.WriteRune(r)
			} else {
				mode = false
				b.WriteByte('V')
				b.WriteRune(x)
			}
		default:
			if r, ok := typexKeyboardRev[x]; ok {
				mode = true
				b.WriteByte('Z')
				b.WriteRune(r)
			} else {
				b.WriteRune(x)
			}
		}
	}
	return b.String()
}

// typexKeyboardDecode reverses typexKeyboardEncode after deciphering (Decrypt
// direction), turning shift sequences back into spaces and symbols.
func typexKeyboardDecode(output string) string {
	var b strings.Builder
	mode := false
	for _, x := range output {
		switch {
		case x == 'X':
			b.WriteByte(' ')
		case x == 'V':
			mode = false
		case x == 'Z':
			mode = true
		case mode:
			if r, ok := typexKeyboardFwd[x]; ok {
				b.WriteRune(r)
			} else {
				// Matches CyberChef, where KEYBOARD[x] is undefined here.
				b.WriteString("undefined")
			}
		default:
			b.WriteRune(x)
		}
	}
	return b.String()
}

// typexMachine holds the rotors (right-to-left), reflector, Rotor-based plugboard
// and keyboard mode.
type typexMachine struct {
	rotors    []*enigmaRotor
	rotorsRev []*enigmaRotor
	reflector *enigmaReflector
	plugboard *enigmaRotor
	keyboard  string
}

// newTypexMachine builds a machine from five rotors ordered right-to-left.
func newTypexMachine(rotors []*enigmaRotor, reflector *enigmaReflector, plugboard *enigmaRotor, keyboard string) *typexMachine {
	rev := make([]*enigmaRotor, len(rotors))
	for i, r := range rotors {
		rev[len(rotors)-1-i] = r
	}
	return &typexMachine{rotors: rotors, rotorsRev: rev, reflector: reflector, plugboard: plugboard, keyboard: keyboard}
}

// step advances the rotors; the two right-hand rotors are static, so stepping
// starts at index 2. Includes the Enigma double-stepping anomaly.
func (m *typexMachine) step() {
	r0, r1 := m.rotors[2], m.rotors[3]
	r0.step()
	if r0.steps[r0.pos] || r1.steps[mod26(r1.pos+1, 26)] {
		r1.step()
		if r1.steps[r1.pos] {
			m.rotors[4].step()
		}
	}
}

// cryptCore enciphers/deciphers from the current state. It matches the Enigma
// path except the plugboard is a rotor: forward on the way in, reverse on the way
// out (Enigma's plugboard is symmetric, so it uses transform both ways).
func (m *typexMachine) cryptCore(input string) string {
	var b strings.Builder
	for _, c := range input {
		letter := enigLetterIndex(c)
		if letter == -1 {
			b.WriteRune(c)
			continue
		}
		m.step()
		letter = m.plugboard.transform(letter)
		for _, r := range m.rotors {
			letter = r.transform(letter)
		}
		letter = m.reflector.transform(letter)
		for _, r := range m.rotorsRev {
			letter = r.revTransform(letter)
		}
		letter = m.plugboard.revTransform(letter)
		b.WriteByte(byte('A' + letter)) // #nosec G115 -- letter is a rotor output in [0,25], so 'A'+letter is a letter
	}
	return b.String()
}

// crypt runs the machine with the Typex keyboard emulation wrapped around the
// core cipher.
func (m *typexMachine) crypt(input string) string {
	inputMod := input
	if m.keyboard == "Encrypt" {
		inputMod = typexKeyboardEncode(input)
	}
	output := m.cryptCore(inputMod)
	if m.keyboard == "Decrypt" {
		output = typexKeyboardDecode(output)
	}
	return output
}
