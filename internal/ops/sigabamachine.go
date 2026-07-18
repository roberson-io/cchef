package ops

import "strings"

// SIGABA (ECM Mark II) machine, ported from CyberChef lib/SIGABA.mjs.

// sigabaCRExamples are the example cipher/control (C/R) rotor wirings; index 0 is
// the editableOption default.
var sigabaCRExamples = []struct{ name, value string }{
	{"Example 1", "SRGWANHPJZFXVIDQCEUKBYOLMT"},
	{"Example 2", "THQEFSAZVKJYULBODCPXNIMWRG"},
	{"Example 3", "XDTUYLEVFNQZBPOGIRCSMHWKAJ"},
	{"Example 4", "LOHDMCWUPSTNGVXYFJREQIKBZA"},
	{"Example 5", "ERXWNZQIJYLVOFUMSGHTCKPBDA"},
	{"Example 6", "FQECYHJIOUMDZVPSLKRTGWXBAN"},
	{"Example 7", "TBYIUMKZDJSOPEWXVANHLCFQGR"},
	{"Example 8", "QZUPDTFNYIAOMLEBWJXCGHKRSV"},
	{"Example 9", "CZWNHEMPOVXLKRSIDGJFYBTQAU"},
	{"Example 10", "ENPXJVKYQBFZTICAGMOHWRLDUS"},
}

// sigabaIExamples are the example index (I) rotor wirings; index 0 is the default.
var sigabaIExamples = []struct{ name, value string }{
	{"Example 1", "6201348957"},
	{"Example 2", "6147253089"},
	{"Example 3", "8239647510"},
	{"Example 4", "7194835260"},
	{"Example 5", "4873205916"},
}

// sigabaIndexOf returns the position of v in s, or -1.
func sigabaIndexOf(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// sigabaRotor is the base rotor shared by cipher/control (C/R) and index (I)
// rotors. numMapping is the (possibly reversed) wiring; posMapping tracks the
// rotation of the rotor's positions.
type sigabaRotor struct {
	state      int
	numMapping []int
	posMapping []int
}

// newSigabaRotor builds a rotor from a numeric wireSetting, starting position and
// orientation.
func newSigabaRotor(wireSetting []int, key int, rev bool) *sigabaRotor {
	r := &sigabaRotor{state: key}
	r.numMapping = sigabaNumMapping(wireSetting, rev)
	r.posMapping = sigabaPosMapping(key, len(r.numMapping), rev)
	return r
}

// sigabaNumMapping returns the wiring, inverted when the rotor is reversed.
func sigabaNumMapping(wireSetting []int, rev bool) []int {
	if !rev {
		return wireSetting
	}
	tmp := make([]int, len(wireSetting))
	for i, w := range wireSetting {
		tmp[w] = i
	}
	return tmp
}

// sigabaPosMapping returns the initial position mapping for a starting state,
// ascending when forward and descending when reversed (mod length).
func sigabaPosMapping(state, length int, rev bool) []int {
	posMapping := make([]int, 0, length)
	if !rev {
		for i := state; i < state+length; i++ {
			posMapping = append(posMapping, ((i%length)+length)%length)
		}
	} else {
		for i := state; i > state-length; i-- {
			posMapping = append(posMapping, ((i%length)+length)%length)
		}
	}
	return posMapping
}

// cryptNum passes a position through the rotor. leftToRight follows the wiring;
// right-to-left inverts it. An out-of-range position returns -1, reproducing
// CyberChef's JavaScript `undefined` cascade (which surfaces as "@" once mapped
// back to a letter) so non-A-Z input matches upstream instead of panicking.
func (r *sigabaRotor) cryptNum(inputPos int, leftToRight bool) int {
	if inputPos < 0 || inputPos >= len(r.posMapping) {
		return -1
	}
	inpNum := r.posMapping[inputPos]
	var outNum int
	if leftToRight {
		outNum = r.numMapping[inpNum]
	} else {
		outNum = sigabaIndexOf(r.numMapping, inpNum)
	}
	return sigabaIndexOf(r.posMapping, outNum)
}

// step rotates the rotor's position mapping by one (last position moves to front).
func (r *sigabaRotor) step() {
	n := len(r.posMapping)
	last := r.posMapping[n-1]
	copy(r.posMapping[1:], r.posMapping[:n-1])
	r.posMapping[0] = last
	r.state = r.posMapping[0]
}

// newSigabaCRRotor builds a cipher/control rotor from a letter wiring, a letter
// starting position and an orientation.
func newSigabaCRRotor(wiring string, key byte, rev bool) *sigabaRotor {
	wireSetting := make([]int, len(wiring))
	for i := 0; i < len(wiring); i++ {
		wireSetting[i] = int(wiring[i] - 'A')
	}
	return newSigabaRotor(wireSetting, int(key-'A'), rev)
}

// crCrypt passes a letter (as a character code) through a cipher/control rotor,
// returning the enciphered character code. Arithmetic stays in int so an
// out-of-range letter maps to code 64 ("@"), matching CyberChef.
func crCrypt(r *sigabaRotor, code int, leftToRight bool) int {
	return r.cryptNum(code-'A', leftToRight) + 'A'
}

// newSigabaIRotor builds an index rotor from a digit wiring and digit start.
func newSigabaIRotor(wiring string, key byte) *sigabaRotor {
	wireSetting := make([]int, len(wiring))
	for i := 0; i < len(wiring); i++ {
		wireSetting[i] = int(wiring[i] - '0')
	}
	return newSigabaRotor(wireSetting, int(key-'0'), false)
}

// sigabaMachine holds the three rotor banks.
type sigabaMachine struct {
	cipher []*sigabaRotor // left-to-right
	// control is stored reversed (signals pass right-to-left).
	control []*sigabaRotor
	index   []*sigabaRotor
}

// newSigabaMachine assembles the machine; the control bank is reversed on entry.
func newSigabaMachine(cipher, control, index []*sigabaRotor) *sigabaMachine {
	rev := make([]*sigabaRotor, len(control))
	for i, r := range control {
		rev[len(control)-1-i] = r
	}
	return &sigabaMachine{cipher: cipher, control: rev, index: index}
}

// controlOutputs runs the four control inputs F,G,H,I through the control bank
// and maps the letter outputs to index-rotor input numbers.
func (m *sigabaMachine) controlOutputs() []int {
	crypt := func(in int) int {
		for _, r := range m.control {
			in = crCrypt(r, in, false)
		}
		return in
	}
	outputs := []int{crypt('F'), crypt('G'), crypt('H'), crypt('I')}
	// logicDict maps input number (1-9) to the letters that select it, iterated in
	// ascending key order.
	logic := []struct {
		key     int
		letters string
	}{
		{1, "B"},
		{2, "C"},
		{3, "DE"},
		{4, "FGH"},
		{5, "IJK"},
		{6, "LMNO"},
		{7, "PQRST"},
		{8, "UVWXYZ"},
		{9, "A"},
	}
	var nums []int
	for _, l := range logic {
		for _, o := range outputs {
			if containsCode(l.letters, o) {
				nums = append(nums, l.key)
				break
			}
		}
	}
	return nums
}

// controlStep steps the control rotors: the fast rotor every letter, the medium
// once the fast reaches "O" (offset 14), the slow once the medium also reaches it.
func (m *sigabaMachine) controlStep() {
	mRotor, fRotor, sRotor := m.control[1], m.control[2], m.control[3]
	if fRotor.state == 14 {
		if mRotor.state == 14 {
			sRotor.step()
		}
		mRotor.step()
	}
	fRotor.step()
}

// indexOutputs passes each control output through the index bank.
func (m *sigabaMachine) indexOutputs(controlInputs []int) []int {
	out := make([]int, len(controlInputs))
	for i, inp := range controlInputs {
		for _, r := range m.index {
			inp = r.cryptNum(inp, true)
		}
		out[i] = inp
	}
	return out
}

// cipherStep steps the cipher rotors selected by the index-bank outputs.
func (m *sigabaMachine) cipherStep(indexInputs []int) {
	// logicDict maps cipher rotor index to the index-bank outputs that step it.
	logic := [][2]int{{0, 9}, {7, 8}, {5, 6}, {3, 4}, {1, 2}}
	var toMove []*sigabaRotor
	for key, item := range logic {
		for _, i := range indexInputs {
			if i == item[0] || i == item[1] {
				toMove = append(toMove, m.cipher[key])
				break
			}
		}
	}
	for _, r := range toMove {
		r.step()
	}
}

// step advances the whole machine by one letter.
func (m *sigabaMachine) step() {
	controlOut := m.controlOutputs()
	m.controlStep()
	indexOut := m.indexOutputs(controlOut)
	m.cipherStep(indexOut)
}

// cipherEncrypt passes a letter (character code) left-to-right through the
// cipher bank.
func (m *sigabaMachine) cipherEncrypt(code int) int {
	for _, r := range m.cipher {
		code = crCrypt(r, code, true)
	}
	return code
}

// cipherDecrypt passes a letter (character code) right-to-left through the
// cipher bank.
func (m *sigabaMachine) cipherDecrypt(code int) int {
	for i := len(m.cipher) - 1; i >= 0; i-- {
		code = crCrypt(m.cipher[i], code, false)
	}
	return code
}

// encrypt enciphers a message. Spaces become "Z" and "Z" becomes "X" so spaces
// survive; each letter steps the machine afterwards. Characters are iterated as
// code points, matching CyberChef's `for...of`.
func (m *sigabaMachine) encrypt(msg string) string {
	var out strings.Builder
	for _, r := range msg {
		code := sigabaUpper(int(r))
		switch code {
		case ' ':
			code = 'Z'
		case 'Z':
			code = 'X'
		}
		out.WriteRune(rune(m.cipherEncrypt(code))) // #nosec G115 -- cipher output is a letter code in [64,90] ("@"-"Z")
		m.step()
	}
	return out.String()
}

// decrypt deciphers a message; a decrypted "Z" is restored to a space.
func (m *sigabaMachine) decrypt(msg string) string {
	var out strings.Builder
	for _, r := range msg {
		dec := m.cipherDecrypt(sigabaUpper(int(r)))
		if dec == 'Z' {
			dec = ' '
		}
		out.WriteRune(rune(dec)) // #nosec G115 -- cipher output is a letter code in [64,90] ("@"-"Z") or space
		m.step()
	}
	return out.String()
}

// sigabaUpper upper-cases an ASCII letter code, matching CyberChef's
// convToUpperCase (other codes pass through unchanged).
func sigabaUpper(code int) int {
	if code >= 'a' && code <= 'z' {
		return code - 32
	}
	return code
}

// containsCode reports whether s contains the character code.
func containsCode(s string, code int) bool {
	for i := 0; i < len(s); i++ {
		if int(s[i]) == code {
			return true
		}
	}
	return false
}
