package ops

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Enigma{})
}

// --- Enigma machine core (ported from CyberChef lib/Enigma.mjs) ---
//
// These types (rotor, plugboard, reflector, machine) are the shared foundation
// the Bombe operations also build on.

// enigmaRotorDefs are the standard German military rotors: a wiring string
// optionally followed by "<" and the stepping points.
var enigmaRotorDefs = []struct{ name, value string }{
	{"I", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R"},
	{"II", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F"},
	{"III", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W"},
	{"IV", "ESOVPZJAYQUIRHXLNFTGKDCMWB<K"},
	{"V", "VZBRGITYUPSDNHLXAWMJQOFECK<A"},
	{"VI", "JPGVOUMFYQBENHZRDKASXLICTW<AN"},
	{"VII", "NZJHGRCXMYSWBOUFAIVLPEKQDT<AN"},
	{"VIII", "FKQHTLXOCBJSPDZRAMEWNIUYGV<AN"},
}

// enigmaRotorsFourth are the thin fourth-slot rotors (no stepping).
var enigmaRotorsFourth = []struct{ name, value string }{
	{"Beta", "LEYJVCNIXWPBQMDRTAKZGFUHOS"},
	{"Gamma", "FSOKANUERHMBTIYCWLQPZXVGJD"},
}

// enigmaReflectorDefs are the standard reflectors (13 transposed pairs).
var enigmaReflectorDefs = []struct{ name, value string }{
	{"B", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW"},
	{"C", "AF BV CP DJ EI GO HY KR LZ MX NW TQ SU"},
	{"B Thin", "AE BN CK DQ FU GY HW IJ LO MP RX SZ TV"},
	{"C Thin", "AR BD CO EJ FN GT HK IV LM PW QZ SX UY"},
}

// enigmaLetters is the A-Z option list used for ring settings and positions.
var enigmaLetters = func() []string {
	l := make([]string, 26)
	for i := range 26 {
		l[i] = string(rune('A' + i))
	}
	return l
}()

var (
	enigmaWiringRe = regexp.MustCompile(`^[A-Z]{26}$`)
	enigmaStepsRe  = regexp.MustCompile(`^[A-Z]{0,26}$`)
	enigmaSingleRe = regexp.MustCompile(`^[A-Z]$`)
	enigmaPairRe   = regexp.MustCompile(`^[A-Z]{2}$`)
	enigmaWSRe     = regexp.MustCompile(`\s+`)
	enigmaNonAlpha = regexp.MustCompile(`[^A-Za-z]`)
)

// enigLetterIndex maps a letter to 0..25, case-insensitively, returning -1 for
// any other character (the permissive form of CyberChef a2i).
func enigLetterIndex(r rune) int {
	switch {
	case r >= 'A' && r <= 'Z':
		return int(r - 'A')
	case r >= 'a' && r <= 'z':
		return int(r - 'a')
	default:
		return -1
	}
}

// enigmaRotor is a single rotor with its forward/reverse wiring, stepping points
// (adjusted for the ring setting) and current position.
type enigmaRotor struct {
	fwd   [26]int
	rev   [26]int
	steps map[int]bool
	pos   int
}

// newEnigmaRotor validates and constructs a rotor.
func newEnigmaRotor(wiring, steps, ringSetting, initialPosition string) (*enigmaRotor, error) {
	switch {
	case !enigmaWiringRe.MatchString(wiring):
		return nil, errors.New("Rotor wiring must be 26 unique uppercase letters") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	case !enigmaStepsRe.MatchString(steps):
		return nil, errors.New("Rotor steps must be 0-26 unique uppercase letters") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	case !enigmaSingleRe.MatchString(ringSetting):
		return nil, errors.New("Rotor ring setting must be exactly one uppercase letter") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	case !enigmaSingleRe.MatchString(initialPosition):
		return nil, errors.New("Rotor initial position must be exactly one uppercase letter") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	r := &enigmaRotor{steps: map[int]bool{}}
	var seen [26]bool
	uniq := 0
	for i := range 26 {
		b := int(wiring[i] - 'A')
		r.fwd[i] = b
		r.rev[b] = i
		if !seen[b] {
			seen[b] = true
			uniq++
		}
	}
	if uniq != 26 {
		return nil, errors.New("Rotor wiring must have each letter exactly once") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	rs := int(ringSetting[0] - 'A')
	for i := range len(steps) {
		r.steps[mod26(int(steps[i]-'A')-rs, 26)] = true
	}
	if len(r.steps) != len(steps) {
		return nil, errors.New("Rotor steps must be unique") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	r.pos = mod26(int(initialPosition[0]-'A')-rs, 26)
	return r, nil
}

// copy returns an independent rotor with the same wiring and current position
// (used by the Bombe, which clones the fast rotor for each scrambler).
func (r *enigmaRotor) copy() *enigmaRotor {
	c := *r
	return &c
}

func (r *enigmaRotor) step() { r.pos = mod26(r.pos+1, 26) }

func (r *enigmaRotor) transform(c int) int { return mod26(r.fwd[mod26(c+r.pos, 26)]-r.pos, 26) }

func (r *enigmaRotor) revTransform(c int) int { return mod26(r.rev[mod26(c+r.pos, 26)]-r.pos, 26) }

// enigmaPairMap is the shared plugboard/reflector swap map.
type enigmaPairMap struct{ m map[int]int }

// newEnigmaPairMap parses whitespace-separated uppercase letter pairs. name is
// used in error messages ("Plugboard"/"Reflector").
func newEnigmaPairMap(pairs, name string) (*enigmaPairMap, error) {
	pm := &enigmaPairMap{m: map[int]int{}}
	if pairs == "" {
		return pm, nil
	}
	for _, pair := range enigmaWSRe.Split(pairs, -1) {
		if !enigmaPairRe.MatchString(pair) {
			return nil, errors.New(name + " must be a whitespace-separated list of uppercase letter pairs") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		a, b := int(pair[0]-'A'), int(pair[1]-'A')
		if a == b { // self-stecker
			continue
		}
		if _, ok := pm.m[a]; ok {
			return nil, fmt.Errorf("%s connects %c more than once", name, pair[0]) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		if _, ok := pm.m[b]; ok {
			return nil, fmt.Errorf("%s connects %c more than once", name, pair[1]) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		pm.m[a] = b
		pm.m[b] = a
	}
	return pm, nil
}

func (pm *enigmaPairMap) transform(c int) int {
	if v, ok := pm.m[c]; ok {
		return v
	}
	return c
}

// enigmaReflector is a PairMap that must cover every letter. It retains the
// original pairs string (used by the Multiple Bombe output).
type enigmaReflector struct {
	m     [26]int
	pairs string
}

func newEnigmaReflector(pairs string) (*enigmaReflector, error) {
	pm, err := newEnigmaPairMap(pairs, "Reflector")
	if err != nil {
		return nil, err
	}
	if len(pm.m) != 26 {
		return nil, errors.New("Reflector must have exactly 13 pairs covering every letter") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	r := &enigmaReflector{pairs: pairs}
	for k, v := range pm.m {
		r.m[k] = v
	}
	return r, nil
}

func (r *enigmaReflector) transform(c int) int { return r.m[c] }

// enigmaMachine holds the rotors (right-to-left), reflector and plugboard.
type enigmaMachine struct {
	rotors    []*enigmaRotor
	rotorsRev []*enigmaRotor
	reflector *enigmaReflector
	plugboard *enigmaPairMap
}

// newEnigmaMachine builds a machine from rotors ordered right-to-left. The
// caller supplies exactly 3 or 4 rotors (the Enigma op guarantees this from its
// model selector), so unlike CyberChef's constructor there is no rotor-count
// guard to reach here.
func newEnigmaMachine(rotors []*enigmaRotor, reflector *enigmaReflector, plugboard *enigmaPairMap) *enigmaMachine {
	rev := make([]*enigmaRotor, len(rotors))
	for i, r := range rotors {
		rev[len(rotors)-1-i] = r
	}
	return &enigmaMachine{rotors: rotors, rotorsRev: rev, reflector: reflector, plugboard: plugboard}
}

// step advances the rotors, including the double-stepping anomaly. The fourth
// rotor (if present) never steps.
func (m *enigmaMachine) step() {
	r0, r1 := m.rotors[0], m.rotors[1]
	r0.step()
	if r0.steps[r0.pos] || r1.steps[mod26(r1.pos+1, 26)] {
		r1.step()
		if r1.steps[r1.pos] {
			m.rotors[2].step()
		}
	}
}

// crypt enciphers (or deciphers) the input from the machine's current state,
// passing non-alphabetic characters through unchanged.
func (m *enigmaMachine) crypt(input string) string {
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
		letter = m.plugboard.transform(letter)
		b.WriteByte(byte('A' + letter)) // #nosec G115 -- letter is a rotor output in [0,25], so 'A'+letter is in [65,90]
	}
	return b.String()
}

// parseRotorStr splits a rotor spec into its wiring and stepping parts. The
// error always names "Rotor 1", matching CyberChef (which passes a constant).
func parseRotorStr(rotor string) (wiring, steps string, err error) {
	if rotor == "" {
		return "", "", errors.New("Rotor 1 must be provided.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if !strings.Contains(rotor, "<") {
		return rotor, "", nil
	}
	// JS `split("<", 2)` keeps only the first two fields, discarding any trailing
	// "<"-separated remainder.
	parts := strings.Split(rotor, "<")
	return parts[0], parts[1], nil
}

// enigmaGroup5 inserts a space after every five characters, except at the very
// end (mirrors CyberChef's /([A-Z]{5})(?!$)/g replacement without RE2 lookahead).
func enigmaGroup5(s string) string {
	var b strings.Builder
	for i := range len(s) {
		b.WriteByte(s[i])
		if (i+1)%5 == 0 && i+1 != len(s) {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// Enigma emulates the WW2 Enigma machine.
type Enigma struct{}

// Meta returns the operation metadata.
func (Enigma) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Enigma",
		Module:      "Bletchley",
		Description: "Encipher/decipher with the WW2 Enigma machine.<br><br>Enigma was used by the German military, among others, around the WW2 era as a portable cipher machine to protect sensitive military, diplomatic and commercial communications.<br><br>The standard set of German military rotors and reflectors are provided. To configure the plugboard, enter a string of connected pairs of letters, e.g. <code>AB CD EF</code> connects A to B, C to D, and E to F. This is also used to create your own reflectors. To create your own rotor, enter the letters that the rotor maps A to Z to, in order, optionally followed by <code>&lt;</code> then a list of stepping points.<br>This is deliberately fairly permissive with rotor placements etc compared to a real Enigma (on which, for example, a four-rotor Enigma uses only the thin reflectors and the beta or gamma rotor in the 4th slot).<br><br>More detailed descriptions of the Enigma, Typex and Bombe operations <a href='https://github.com/gchq/CyberChef/wiki/Enigma,-the-Bombe,-and-Typex'>can be found here</a>.",
		InfoURL:     "https://wikipedia.org/wiki/Enigma_machine",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Enigma) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Model", Type: core.ArgOption, Value: []string{"3-rotor", "4-rotor"}},
		{Name: "Left-most (4th) rotor", Type: core.ArgEditableOption, Value: enigmaRotorsFourth[0].value},
		{Name: "Left-most rotor ring setting", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Left-most rotor initial value", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Left-hand rotor", Type: core.ArgEditableOption, Value: enigmaRotorDefs[0].value},
		{Name: "Left-hand rotor ring setting", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Left-hand rotor initial value", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Middle rotor", Type: core.ArgEditableOption, Value: enigmaRotorDefs[1].value},
		{Name: "Middle rotor ring setting", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Middle rotor initial value", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Right-hand rotor", Type: core.ArgEditableOption, Value: enigmaRotorDefs[2].value},
		{Name: "Right-hand rotor ring setting", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Right-hand rotor initial value", Type: core.ArgOption, Value: enigmaLetters},
		{Name: "Reflector", Type: core.ArgEditableOption, Value: enigmaReflectorDefs[0].value},
		{Name: "Plugboard", Type: core.ArgString, Value: ""},
		{Name: "Strict output", Type: core.ArgBoolean, Value: true},
	}
}

// Run performs the encryption/decryption. Ported from CyberChef Enigma.mjs.
func (Enigma) Run(in *core.Dish, args []any) (*core.Dish, error) {
	model := args[0].(string)
	removeOther := args[15].(bool)

	var rotors []*enigmaRotor
	for i := range 4 {
		if i == 0 && model == "3-rotor" {
			continue // skip the 4th-rotor settings
		}
		wiring, steps, err := parseRotorStr(args[i*3+1].(string))
		if err != nil {
			return nil, err
		}
		rotor, err := newEnigmaRotor(wiring, steps, args[i*3+2].(string), args[i*3+3].(string))
		if err != nil {
			return nil, err
		}
		rotors = append(rotors, rotor)
	}
	// Rotors are handled right-to-left.
	for i, j := 0, len(rotors)-1; i < j; i, j = i+1, j-1 {
		rotors[i], rotors[j] = rotors[j], rotors[i]
	}
	reflector, err := newEnigmaReflector(args[13].(string))
	if err != nil {
		return nil, err
	}
	plugboard, err := newEnigmaPairMap(args[14].(string), "Plugboard")
	if err != nil {
		return nil, err
	}

	input := in.String()
	if removeOther {
		input = enigmaNonAlpha.ReplaceAllString(input, "")
	}
	result := newEnigmaMachine(rotors, reflector, plugboard).crypt(input)
	if removeOther {
		result = enigmaGroup5(result)
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}
