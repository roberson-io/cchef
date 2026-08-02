package ops

// Lorenz SZ40/42 machine.

// Chi (X1-5) and Psi (S1-5) wheel sizes, indexed 0-based.
var (
	lorenzChiSizes = [5]int{41, 31, 29, 26, 23}
	lorenzPsiSizes = [5]int{43, 47, 51, 53, 59}
)

// lorenzMachine holds the running wheel state for one encipher pass.
type lorenzMachine struct {
	model  string
	kt     bool
	mode   string
	format string

	chi map[int][]int // 1..5
	psi map[int][]int // 1..5
	mu  map[int][]int // 1..2 (M61, M37)

	x   [5]int // Chi wheel positions (X1-5)
	s   [5]int // Psi wheel positions (S1-5)
	m37 int
	m61 int

	m61lug int
	m37lug int
	p5     [3]int
}

// newLorenzMachine initialises the machine from the config, wheel settings and
// rotor start positions.
func newLorenzMachine(cfg lorenzConfig, settings lorenzRings, args []any) *lorenzMachine {
	m := &lorenzMachine{
		model:  cfg.model,
		kt:     cfg.kt,
		mode:   cfg.mode,
		format: cfg.format,
		chi:    settings.X,
		psi:    settings.S,
		mu:     settings.M,
		m37:    colNum(args[12]),
		m61:    colNum(args[13]),
	}
	m.s = [5]int{colNum(args[7]), colNum(args[8]), colNum(args[9]), colNum(args[10]), colNum(args[11])}
	m.x = [5]int{colNum(args[14]), colNum(args[15]), colNum(args[16]), colNum(args[17]), colNum(args[18])}
	m.m61lug = m.mu[1][m.m61-1]
	m.m37lug = m.mu[2][m.m37-1]
	return m
}

// encipherAll processes each character of the ITA2 input, returning the
// enciphered ITA2 output.
func (m *lorenzMachine) encipherAll(ita2Input string) string {
	var out []byte
	for i := 0; i < len(ita2Input); i++ {
		if r, ok := m.encipher(ita2Input[i]); ok {
			out = append(out, r)
		}
	}
	return string(out)
}

// encipher enciphers one ITA2 character: XOR the impulse bits with the current
// Chi and Psi wheel bits, then step the wheels. Returns ok=false for a character
// not present in the ITA2 table (skipped, matching the JS map returning "").
func (m *lorenzMachine) encipher(letter byte) (byte, bool) {
	// letter is already upper-cased by lorenzToITA2 (which validates/normalises
	// the whole stream), so no per-character upper-casing is needed here.
	x2bptr := m.x[1] + 1
	if x2bptr == 32 {
		x2bptr = 1
	}
	s1bptr := m.s[0] + 1
	if s1bptr == 44 {
		s1bptr = 1
	}

	thisChi := [5]int{m.chi[1][m.x[0]-1], m.chi[2][m.x[1]-1], m.chi[3][m.x[2]-1], m.chi[4][m.x[3]-1], m.chi[5][m.x[4]-1]}
	thisPsi := [5]int{m.psi[1][m.s[0]-1], m.psi[2][m.s[1]-1], m.psi[3][m.s[2]-1], m.psi[4][m.s[3]-1], m.psi[5][m.s[4]-1]}

	bits, ok := ita2Table[letter]
	if !ok {
		return 0, false
	}

	var xor [5]byte
	for i := range 5 {
		// XOR of three 0/1 bits offset from '0' is always the byte '0' or '1'.
		xor[i] = byte('0' + (int(bits[i]-'0') ^ thisPsi[i] ^ thisChi[i])) // #nosec G115 -- result is '0' or '1'
	}

	m.stepChi()
	if m.m61--; m.m61 < 1 {
		m.m61 = 61
	}
	if m.m61lug == 1 {
		if m.m37--; m.m37 < 1 {
			m.m37 = 37
		}
	}

	basicmotor := m.m37lug
	m.p5[2] = m.p5[1]
	m.p5[1] = m.p5[0]
	if m.mode == "Send" {
		m.p5[0] = int(bits[4] - '0')
	} else {
		m.p5[0] = int(xor[4] - '0')
	}

	if m.totalMotor(basicmotor, x2bptr, s1bptr) == 1 {
		m.stepPsi()
	}

	m.m61lug = m.mu[1][m.m61-1]
	m.m37lug = m.mu[2][m.m37-1]

	rtn := lorenzReverseITA2[string(xor[:])]
	if m.format == "5/8/9" {
		switch rtn {
		case "+":
			rtn = "5"
		case "-":
			rtn = "8"
		case ".":
			rtn = "9"
		}
	}
	return rtn[0], true
}

// totalMotor computes whether the Psi wheels move this character, applying the
// model-specific limitation (and the KT-Schalter for the SZ42 models).
func (m *lorenzMachine) totalMotor(basicmotor, x2bptr, s1bptr int) int {
	switch m.model {
	case "SZ40":
		return basicmotor
	case "SZ42a":
		lim := m.chi[2][x2bptr-1]
		lim = m.applyKT(lim)
		return motorFromLimit(basicmotor, lim)
	case "SZ42b":
		lim := 1
		if m.chi[2][x2bptr-1] == m.psi[1][s1bptr-1] {
			lim = 0
		}
		lim = m.applyKT(lim)
		return motorFromLimit(basicmotor, lim)
	default:
		// Unreachable: Model is a fixed option list validated by CoerceArgs.
		return basicmotor
	}
}

// applyKT applies the KT-Schalter adjustment: when set, the limitation is
// compared against the impulse-5 value from two characters back.
func (m *lorenzMachine) applyKT(lim int) int {
	if !m.kt {
		return lim
	}
	if lim == m.p5[2] {
		return 0
	}
	return 1
}

// motorFromLimit: no move when the basic motor is off and the limitation is on.
func motorFromLimit(basicmotor, lim int) int {
	if basicmotor == 0 && lim == 1 {
		return 0
	}
	return 1
}

// stepChi advances (decrements, wrapping) the five Chi wheels one position.
func (m *lorenzMachine) stepChi() {
	for i := range 5 {
		if m.x[i]--; m.x[i] < 1 {
			m.x[i] = lorenzChiSizes[i]
		}
	}
}

// stepPsi advances (decrements, wrapping) the five Psi wheels one position.
func (m *lorenzMachine) stepPsi() {
	for i := range 5 {
		if m.s[i]--; m.s[i] < 1 {
			m.s[i] = lorenzPsiSizes[i]
		}
	}
}
