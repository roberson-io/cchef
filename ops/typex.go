package ops

import (
	"regexp"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Typex{})
}

// typexRotorLabels name the five rotor-selection args (with the hand/static hints).
var typexRotorLabels = []string{
	"1st (left-hand) rotor", "2nd rotor", "3rd (middle) rotor",
	"4th (static) rotor", "5th (right-hand, static) rotor",
}

// typexOrdinals prefix the reversed/ring/initial args of each rotor.
var typexOrdinals = []string{"1st", "2nd", "3rd", "4th", "5th"}

// typexStripEncrypt keeps only the characters the Typex keyboard can encrypt
// (letters, digits, space and the supported symbols).
var typexStripEncrypt = regexp.MustCompile(`[^A-Za-z0-9 /%£()',.-]`)

// Typex emulates the WW2 Typex machine.
type Typex struct{}

// Meta returns the operation metadata.
func (Typex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Typex",
		Module:      "Bletchley",
		Description: "Encipher/decipher with the WW2 Typex machine.<br><br>Typex was originally built by the British Royal Air Force prior to WW2, and is based on the Enigma machine with some improvements made, including using five rotors with more stepping points and interchangeable wiring cores. It was used across the British and Commonwealth militaries. A number of later variants were produced; here we simulate a WW2 era Mark 22 Typex with plugboards for the reflector and input. Typex rotors were changed regularly and none are public: a random example set are provided.<br><br>To configure the reflector plugboard, enter a string of connected pairs of letters in the reflector box, e.g. <code>AB CD EF</code> connects A to B, C to D, and E to F (you'll need to connect every letter). There is also an input plugboard: unlike Enigma's plugboard, it's not restricted to pairs, so it's entered like a rotor (without stepping). To create your own rotor, enter the letters that the rotor maps A to Z to, in order, optionally followed by <code>&lt;</code> then a list of stepping points.<br><br>More detailed descriptions of the Enigma, Typex and Bombe operations <a href='https://github.com/gchq/CyberChef/wiki/Enigma,-the-Bombe,-and-Typex'>can be found here</a>.",
		InfoURL:     "https://wikipedia.org/wiki/Typex",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions (five rotors of four args each, then the
// reflector, plugboard, keyboard mode and strict-output flag).
func (Typex) Args() []core.ArgDef {
	args := make([]core.ArgDef, 0, 24)
	for i := range 5 {
		args = append(
			args,
			core.ArgDef{Name: typexRotorLabels[i], Type: core.ArgEditableOption, Value: typexRotors[i].value},
			core.ArgDef{Name: typexOrdinals[i] + " rotor reversed", Type: core.ArgBoolean, Value: false},
			core.ArgDef{Name: typexOrdinals[i] + " rotor ring setting", Type: core.ArgOption, Value: enigmaLetters},
			core.ArgDef{Name: typexOrdinals[i] + " rotor initial value", Type: core.ArgOption, Value: enigmaLetters},
		)
	}
	return append(
		args,
		core.ArgDef{Name: "Reflector", Type: core.ArgEditableOption, Value: typexReflectors[0].value},
		core.ArgDef{Name: "Plugboard", Type: core.ArgString, Value: ""},
		core.ArgDef{Name: "Typex keyboard emulation", Type: core.ArgOption, Value: []string{"None", "Encrypt", "Decrypt"}},
		core.ArgDef{Name: "Strict output", Type: core.ArgBoolean, Value: true},
	)
}

// Run performs the encryption/decryption.
func (Typex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	reflectorStr := args[20].(string)
	plugboardStr := args[21].(string)
	keyboard := args[22].(string)
	removeOther := args[23].(bool)

	var rotors []*enigmaRotor
	for i := range 5 {
		wiring, steps, err := typexParseRotorStr(args[i*4].(string))
		if err != nil {
			return nil, err
		}
		rotor, err := newTypexRotor(wiring, steps, args[i*4+1].(bool), args[i*4+2].(string), args[i*4+3].(string))
		if err != nil {
			return nil, err
		}
		rotors = append(rotors, rotor)
	}
	// Rotors are handled right-to-left.
	for i, j := 0, len(rotors)-1; i < j; i, j = i+1, j-1 {
		rotors[i], rotors[j] = rotors[j], rotors[i]
	}
	reflector, err := newEnigmaReflector(reflectorStr)
	if err != nil {
		return nil, err
	}
	plugboardMod := plugboardStr
	if plugboardMod == "" {
		plugboardMod = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}
	plugboard, err := newTypexPlugboard(plugboardMod)
	if err != nil {
		return nil, err
	}

	input := in.String()
	if removeOther {
		if keyboard == "Encrypt" {
			input = typexStripEncrypt.ReplaceAllString(input, "")
		} else {
			input = enigmaNonAlpha.ReplaceAllString(input, "")
		}
	}
	result := newTypexMachine(rotors, reflector, plugboard, keyboard).crypt(input)
	if removeOther && keyboard != "Decrypt" {
		result = enigmaGroup5(result)
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}
