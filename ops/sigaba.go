package ops

import (
	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(SIGABA{})
}

// sigabaLongOrdinals label a rotor by position (with the hand/middle hints).
var sigabaLongOrdinals = []string{"1st (left-hand)", "2nd", "3rd (middle)", "4th", "5th (right-hand)"}

// sigabaShortOrdinals label the reversed/initial-value args (no hand/middle hint).
var sigabaShortOrdinals = []string{"1st", "2nd", "3rd", "4th", "5th"}

// sigabaLetters is A-Z; sigabaNumbers is 0-9 (rotor initial-value choices).
var sigabaLetters = func() []string {
	l := make([]string, 26)
	for i := range 26 {
		l[i] = string(rune('A' + i))
	}
	return l
}()

var sigabaNumbers = func() []string {
	n := make([]string, 10)
	for i := range 10 {
		n[i] = string(rune('0' + i))
	}
	return n
}()

// SIGABA emulates the WW2 SIGABA (ECM Mark II) cipher machine.
type SIGABA struct{}

// Meta returns the operation metadata.
func (SIGABA) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SIGABA",
		Module:      "Bletchley",
		Description: "Encipher/decipher with the WW2 SIGABA machine. <br><br>SIGABA, otherwise known as ECM Mark II, was used by the United States for message encryption during WW2 up to the 1950s. It was developed in the 1930s by the US Army and Navy, and has up to this day never been broken. Consisting of 15 rotors: 5 cipher rotors and 10 rotors (5 control rotors and 5 index rotors) controlling the stepping of the cipher rotors, the rotor stepping for SIGABA is much more complex than other rotor machines of its time, such as Enigma. All example rotor wirings are random example sets.<br><br>To configure rotor wirings, for the cipher and control rotors enter a string of letters which map from A to Z, and for the index rotors enter a sequence of numbers which map from 0 to 9. Note that encryption is not the same as decryption, so first choose the desired mode. <br><br> Note: Whilst this has been tested against other software emulators, it has not been tested against hardware.",
		InfoURL:     "https://wikipedia.org/wiki/SIGABA",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions (5 cipher, 5 control, 5 index rotors,
// then the mode) in the same order as CyberChef's SIGABA.mjs.
func (SIGABA) Args() []core.ArgDef {
	args := make([]core.ArgDef, 0, 41)
	crDefault := sigabaCRExamples[0].value
	iDefault := sigabaIExamples[0].value
	for _, kind := range []string{"cipher", "control"} {
		for i := range 5 {
			args = append(
				args,
				core.ArgDef{Name: sigabaLongOrdinals[i] + " " + kind + " rotor", Type: core.ArgEditableOption, Value: crDefault},
				core.ArgDef{Name: sigabaShortOrdinals[i] + " " + kind + " rotor reversed", Type: core.ArgBoolean, Value: false},
				core.ArgDef{Name: sigabaShortOrdinals[i] + " " + kind + " rotor initial value", Type: core.ArgOption, Value: sigabaLetters},
			)
		}
	}
	for i := range 5 {
		args = append(
			args,
			core.ArgDef{Name: sigabaLongOrdinals[i] + " index rotor", Type: core.ArgEditableOption, Value: iDefault},
			core.ArgDef{Name: sigabaShortOrdinals[i] + " index rotor initial value", Type: core.ArgOption, Value: sigabaNumbers},
		)
	}
	args = append(args, core.ArgDef{Name: "SIGABA mode", Type: core.ArgOption, Value: []string{"Encrypt", "Decrypt"}})
	return args
}

// Run enciphers or deciphers with SIGABA. Ported from CyberChef SIGABA.mjs.
func (SIGABA) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cipher := make([]*sigabaRotor, 5)
	control := make([]*sigabaRotor, 5)
	index := make([]*sigabaRotor, 5)
	for i := range 5 {
		cipher[i] = newSigabaCRRotor(args[i*3].(string), args[i*3+2].(string)[0], args[i*3+1].(bool))
	}
	for i := 5; i < 10; i++ {
		control[i-5] = newSigabaCRRotor(args[i*3].(string), args[i*3+2].(string)[0], args[i*3+1].(bool))
	}
	for i := 15; i < 20; i++ {
		index[i-15] = newSigabaIRotor(args[i*2].(string), args[i*2+1].(string)[0])
	}
	machine := newSigabaMachine(cipher, control, index)

	var result string
	if args[40].(string) == "Decrypt" {
		result = machine.decrypt(in.String())
	} else {
		result = machine.encrypt(in.String())
	}
	return core.NewDish([]byte(result), core.TypeString), nil
}
