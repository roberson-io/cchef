package ops

import (
	"fmt"
	"strconv"
	"strings"
)

// Colossus computer, ported from CyberChef src/core/lib/Colossus.mjs
// (VirtualColossus, Crown Copyright 2019, Apache-2.0).

// colossusCondRow is one conditional (top-section) Q-bus programming row.
type colossusCondRow struct {
	Qswitches [5]string // each "", "." or "x"
	Negate    bool
	Counter   string // "", "1".."5"
}

// colossusAddRow is the single addition (bottom-section) Q-bus row.
type colossusAddRow struct {
	Qswitches [5]bool
	Equals    string // "", "." or "x"
	C1        bool
}

// colossusSwitches holds the full K-rack switch programming.
type colossusSwitches struct {
	condition     [3]colossusCondRow
	condNegateAll bool
	addition      colossusAddRow
	addNegateAll  bool
	totalMotor    string
}

// colossusLimit selects which stepping limitations are active.
type colossusLimit struct{ X2, S1, P5 bool }

// The five wheels (Chi X1-5, Psi S1-5) and two motors index rotorPtrs by name.
var colRotorKeys = []string{"X1", "X2", "X3", "X4", "X5", "M61", "M37", "S1", "S2", "S3", "S4", "S5"}

// colossusComputer holds the running machine state for one recipe execution.
type colossusComputer struct {
	rings      lorenzRings // reversed pattern rings, indexed 1-based like the source
	ciphertext string
	qbusZ      string
	qbusChi    string
	qbusPsi    string
	qbus       colossusSwitches
	fast       string
	slow       string
	starts     map[string]int
	rotorPtrs  map[string]int
	settotal   int
	limit      colossusLimit

	allCounters [5]int

	Zbits        [5]int
	ZbitsOneBack [5]int
	Qbits        [5]int

	Xbits        [5]int
	Xptr         [5]int
	XbitsOneBack [5]int

	Sbits        [5]int
	Sptr         [5]int
	SbitsOneBack [5]int

	Mptr [2]int // [0]=M37, [1]=M61

	totalmotor int
	P5Zbit     [2]int
}

// newColossusComputer builds the machine from the coerced argument list.
func newColossusComputer(input string, args []any) *colossusComputer {
	c := &colossusComputer{
		ciphertext: input,
		qbusZ:      args[2].(string),
		qbusChi:    args[3].(string),
		qbusPsi:    args[4].(string),
		fast:       args[43].(string),
		slow:       args[44].(string),
		settotal:   colNum(args[42]),
	}
	c.initThyratrons(args[1].(string))
	c.limit = colossusParseLimit(args[5].(string))
	c.qbus = colossusParseSwitches(args)
	c.starts = colossusParseStarts(args)
	return c
}

// colossusParseLimit maps the Limitation option to its active flags. The tests
// key on the Greek "Χ2"/"Ψ1" exactly as CyberChef does.
func colossusParseLimit(limitation string) colossusLimit {
	// Keyed on the Greek "Χ2"/"Ψ1" exactly as CyberChef: the Latin-X limitation
	// options therefore do not enable the Χ2 limitation.
	return colossusLimit{
		X2: strings.Contains(limitation, "Χ2"),
		S1: strings.Contains(limitation, "Ψ1"),
		P5: strings.Contains(limitation, "P5"),
	}
}

// colossusParseSwitches reads the K-rack switch arguments into structured form.
func colossusParseSwitches(args []any) colossusSwitches {
	sw := colossusSwitches{
		condNegateAll: args[30].(bool),
		addNegateAll:  args[39].(bool),
		totalMotor:    args[40].(string),
	}
	for r := range 3 {
		base := 9 + r*7
		row := colossusCondRow{Negate: args[base+5].(bool), Counter: args[base+6].(string)}
		for q := range 5 {
			row.Qswitches[q] = args[base+q].(string)
		}
		sw.condition[r] = row
	}
	add := colossusAddRow{Equals: args[37].(string), C1: args[38].(bool)}
	for q := range 5 {
		add.Qswitches[q] = args[32+q].(bool)
	}
	sw.addition = add
	return sw
}

// colossusParseStarts reads the rotor start positions into a name→position map.
func colossusParseStarts(args []any) map[string]int {
	return map[string]int{
		"X1": colNum(args[45]), "X2": colNum(args[46]), "X3": colNum(args[47]),
		"X4": colNum(args[48]), "X5": colNum(args[49]),
		"M61": colNum(args[50]), "M37": colNum(args[51]),
		"S1": colNum(args[52]), "S2": colNum(args[53]), "S3": colNum(args[54]),
		"S4": colNum(args[55]), "S5": colNum(args[56]),
	}
}

// initThyratrons loads the selected pattern's wheel bits, reversed as the source does.
func (c *colossusComputer) initThyratrons(pattern string) {
	src := initPatterns[pattern]
	c.rings = lorenzRings{
		X: map[int][]int{}, S: map[int][]int{}, M: map[int][]int{},
	}
	for i := 1; i <= 5; i++ {
		c.rings.X[i] = reverseInts(src.X[i])
		c.rings.S[i] = reverseInts(src.S[i])
	}
	c.rings.M[1] = reverseInts(src.M[1])
	c.rings.M[2] = reverseInts(src.M[2])
}

// reverseInts returns a reversed copy of a.
func reverseInts(a []int) []int {
	out := make([]int, len(a))
	for i, v := range a {
		out[len(a)-1-i] = v
	}
	return out
}

// run executes the full Colossus run, returning the printout, the final counter
// values and the run count.
func (c *colossusComputer) run() (string, [5]int, int) {
	c.rotorPtrs = map[string]int{}
	for _, k := range colRotorKeys {
		c.rotorPtrs[k] = c.starts[k]
	}
	runcount := 1
	var printout strings.Builder
	printout.WriteString(c.fast + " " + c.slow + "\n")

	for {
		c.allCounters = [5]int{}
		c.ZbitsOneBack = [5]int{}
		c.XbitsOneBack = [5]int{}

		c.runTape()
		printout.WriteString(c.printLine())

		if c.fast != "" {
			c.rotorPtrs[c.fast]++
			if c.rotorPtrs[c.fast] > rotorSizes[c.fast] {
				c.rotorPtrs[c.fast] = 1
			}
		}
		if c.slow != "" && c.rotorPtrs[c.fast] == c.starts[c.fast] {
			c.rotorPtrs[c.slow]++
			if c.rotorPtrs[c.slow] > rotorSizes[c.slow] {
				c.rotorPtrs[c.slow] = 1
			}
		}
		runcount++
		if c.rotorsAtStart() {
			break
		}
	}
	return printout.String(), c.allCounters, runcount
}

// rotorsAtStart reports whether every rotor pointer is back at its start.
func (c *colossusComputer) rotorsAtStart() bool {
	for _, k := range colRotorKeys {
		if c.rotorPtrs[k] != c.starts[k] {
			return false
		}
	}
	return true
}

// printLine formats the counter line for the current run (only counters above
// the Set Total are printed), or "" when none qualify.
func (c *colossusComputer) printLine() string {
	fastRef, slowRef := "00", "00"
	if c.fast != "" {
		fastRef = pad2Zero(c.rotorPtrs[c.fast])
	}
	if c.slow != "" {
		slowRef = pad2Zero(c.rotorPtrs[c.slow])
	}
	var printline strings.Builder
	for i := range 5 {
		if c.allCounters[i] > c.settotal {
			printline.WriteString(string(rune('a'+i)) + strconv.Itoa(c.allCounters[i]) + " ")
		}
	}
	if printline.String() == "" {
		return ""
	}
	return fastRef + " " + slowRef + " : " + printline.String() + "\n"
}

// pad2Zero renders n as a minimum-two-digit, zero-padded decimal.
func pad2Zero(n int) string {
	return fmt.Sprintf("%02d", n)
}

// runTape runs one full pass of the cipher tape through the machine.
func (c *colossusComputer) runTape() {
	c.Xptr = [5]int{c.rotorPtrs["X1"], c.rotorPtrs["X2"], c.rotorPtrs["X3"], c.rotorPtrs["X4"], c.rotorPtrs["X5"]}
	c.Mptr = [2]int{c.rotorPtrs["M37"], c.rotorPtrs["M61"]}
	c.Sptr = [5]int{c.rotorPtrs["S1"], c.rotorPtrs["S2"], c.rotorPtrs["S3"], c.rotorPtrs["S4"], c.rotorPtrs["S5"]}

	for i := 0; i < len(c.ciphertext); i++ {
		ch := c.ciphertext[i]
		c.getQbusInputs(ch)
		cnt := c.runQbusProcessingConditional()
		c.runQbusProcessingAddition(&cnt)

		bits := ita2Table[ch]
		c.P5Zbit[1] = c.P5Zbit[0]
		c.P5Zbit[0] = int(bits[4] - '0')

		c.stepThyratrons()
	}
}

// ita2Bits returns the five impulse bits of an ITA2 character as ints.
func ita2Bits(ch byte) [5]int {
	s := ita2Table[ch]
	var out [5]int
	for i := range 5 {
		out[i] = int(s[i] - '0')
	}
	return out
}

// getQbusInputs assembles the Q-bus bits for the current character from the
// selected Z / Chi / Psi inputs (direct or delta).
func (c *colossusComputer) getQbusInputs(ch byte) {
	c.Zbits = ita2Bits(ch)
	switch c.qbusZ {
	case "Z":
		c.Qbits = c.Zbits
	case "ΔZ":
		for b := range 5 {
			c.Qbits[b] = c.Zbits[b] ^ c.ZbitsOneBack[b]
		}
	}
	c.ZbitsOneBack = c.Zbits

	for b := range 5 {
		c.Xbits[b] = c.rings.X[b+1][c.Xptr[b]-1]
	}
	switch c.qbusChi {
	case "Χ":
		for b := range 5 {
			c.Qbits[b] ^= c.Xbits[b]
		}
	case "ΔΧ":
		for b := range 5 {
			c.Qbits[b] ^= c.Xbits[b]
			c.Qbits[b] ^= c.XbitsOneBack[b]
		}
	}
	c.XbitsOneBack = c.Xbits

	for b := range 5 {
		c.Sbits[b] = c.rings.S[b+1][c.Sptr[b]-1]
	}
	switch c.qbusPsi {
	case "Ψ":
		for b := range 5 {
			c.Qbits[b] ^= c.Sbits[b]
		}
	case "ΔΨ":
		for b := range 5 {
			c.Qbits[b] ^= c.Sbits[b]
			c.Qbits[b] ^= c.SbitsOneBack[b]
		}
	}
	c.SbitsOneBack = c.Sbits
}

// tri-state conditional result values, matching the JS -1 / false / true.
const (
	condUnset = -1
	condFalse = 0
	condTrue  = 1
)

// runQbusProcessingConditional evaluates the top-section conditional rows,
// ANDing each active row into its counter column.
func (c *colossusComputer) runQbusProcessingConditional() [5]int {
	cnt := [5]int{condUnset, condUnset, condUnset, condUnset, condUnset}
	for r := range len(c.qbus.condition) {
		row := c.qbus.condition[r]
		if row.Counter == "" {
			continue
		}
		cPnt := int(row.Counter[0] - '1')
		match := true
		qsw := readBusSwitches(row.Qswitches)
		for s := range 5 {
			if qsw[s] >= 0 && qsw[s] != c.Qbits[s] {
				match = false
			}
		}
		if row.Negate {
			match = !match
		}
		if cnt[cPnt] == condUnset {
			cnt[cPnt] = b2cond(match)
		} else if !match {
			cnt[cPnt] = condFalse
		}
	}
	for col := range 5 {
		if c.qbus.condNegateAll && cnt[col] != condUnset {
			cnt[col] = condTrue - cnt[col]
		}
	}
	return cnt
}

// runQbusProcessingAddition applies the bottom-section addition row and then
// increments the machine counters for every set column (subject to negate and
// the total-motor gate).
func (c *colossusComputer) runQbusProcessingAddition(cnt *[5]int) {
	row := c.qbus.addition
	if row.C1 {
		addition := 0
		for s := range 5 {
			if row.Qswitches[s] {
				addition ^= c.Qbits[s]
			}
		}
		equals := colEquals(row.Equals)
		if addition == equals {
			if cnt[0] == condUnset {
				cnt[0] = condTrue
			}
		} else {
			cnt[0] = condFalse
		}
	}
	for col := range 5 {
		if c.qbus.addNegateAll && cnt[col] != condUnset {
			cnt[col] = condTrue - cnt[col]
		}
		if c.totalMotorGate() && cnt[col] == condTrue {
			c.allCounters[col]++
		}
	}
}

// totalMotorGate reports whether counting is enabled for this character given
// the Total Motor switch and the current total-motor bit.
func (c *colossusComputer) totalMotorGate() bool {
	tm := c.qbus.totalMotor
	return tm == "" || (tm == "x" && c.totalmotor == 0) || (tm == "." && c.totalmotor == 1)
}

// colEquals maps the Add-Equals switch to its comparison value (blank=-1).
func colEquals(v string) int {
	switch v {
	case "":
		return -1
	case ".":
		return 0
	default:
		return 1
	}
}

// readBusSwitches converts a conditional row's "."/"x" switches to 0/1 (-1 blank).
func readBusSwitches(row [5]string) [5]int {
	out := [5]int{-1, -1, -1, -1, -1}
	for i := range 5 {
		switch row[i] {
		case ".":
			out[i] = 0
		case "x":
			out[i] = 1
		}
	}
	return out
}

// b2cond maps a Go bool to the tri-state true/false constant.
func b2cond(b bool) int {
	if b {
		return condTrue
	}
	return condFalse
}

// stepThyratrons advances the wheels one character: Chi wheels always step, the
// Psi wheels step only when the total motor is set, and the motor wheels step
// per the M61→M37 relationship. It also computes the total-motor bit used by the
// next character, applying the selected limitations.
func (c *colossusComputer) stepThyratrons() {
	lim := c.limitationBit()

	basicmotor := c.rings.M[2][c.Mptr[0]-1]
	c.totalmotor = basicmotor
	if c.limit.X2 || c.limit.S1 {
		if basicmotor == 0 && lim == 1 {
			c.totalmotor = 0
		} else {
			c.totalmotor = 1
		}
	}

	c.stepWheels()
}

// limitationBit computes the limitation value used to constrain the total motor,
// combining the selected Χ2 / Ψ1 / P5 limitations from the relevant "one back"
// wheel positions.
func (c *colossusComputer) limitationBit() int {
	lim := 1
	if c.limit.X2 {
		x2b := stepBack(c.Xptr[1], rotorSizes["X2"])
		lim = c.rings.X[2][x2b-1]
	}
	if c.limit.S1 {
		s1b := stepBack(c.Sptr[0], rotorSizes["S1"])
		lim ^= c.rings.S[1][s1b-1]
	}
	if c.limit.P5 {
		x5b := stepBack(stepBack(c.Xptr[4], rotorSizes["X5"]), rotorSizes["X5"])
		s5b := stepBack(stepBack(c.Sptr[4], rotorSizes["S5"]), rotorSizes["S5"])
		p5lim := c.P5Zbit[1]
		p5lim ^= c.rings.X[5][x5b-1]
		p5lim ^= c.rings.S[5][s5b-1]
		lim ^= p5lim
	}
	return lim
}

// stepWheels advances the Chi wheels (always), the Psi wheels (only when the
// total motor is set) and the two motor wheels (M37 gated by M61), each wrapping
// at its rotor size.
func (c *colossusComputer) stepWheels() {
	for r := range 5 {
		c.Xptr[r]++
		if c.Xptr[r] > rotorSizes[fmt.Sprintf("X%d", r+1)] {
			c.Xptr[r] = 1
		}
	}
	if c.totalmotor != 0 {
		for r := range 5 {
			c.Sptr[r]++
			if c.Sptr[r] > rotorSizes[fmt.Sprintf("S%d", r+1)] {
				c.Sptr[r] = 1
			}
		}
	}
	if c.rings.M[1][c.Mptr[1]-1] == 1 {
		c.Mptr[0]++
	}
	if c.Mptr[0] > rotorSizes["M37"] {
		c.Mptr[0] = 1
	}
	c.Mptr[1]++
	if c.Mptr[1] > rotorSizes["M61"] {
		c.Mptr[1] = 1
	}
}

// stepBack returns the previous pointer position, wrapping 1 → size, as the
// source's "one back" calculations do.
func stepBack(ptr, size int) int {
	p := ptr - 1
	if p == 0 {
		p = size
	}
	return p
}
