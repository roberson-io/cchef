package ops

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(Bombe{})
	core.Register(MultipleBombe{})
}

// --- Bombe machine core (ported from CyberChef lib/Bombe.mjs) ---
//
// The Bombe attacks Enigma using a "crib" (known plaintext). It builds a menu
// graph from crib↔ciphertext letter mappings, then for every rotor setting runs
// an electrical simulation (with Welchman's diagonal board) to rule out
// steckering hypotheses; settings that survive are candidate stops.

// bombeNode is a menu-graph node representing a plain/ciphertext letter.
type bombeNode struct {
	letter  byte
	edges   []*bombeEdge // insertion-ordered (JS Set)
	visited bool
}

// bombeEdge is an Enigma transformation between two letters at a rotor position.
type bombeEdge struct {
	pos          int
	node1, node2 *bombeNode
	visited      bool
}

func newBombeEdge(pos int, n1, n2 *bombeNode) *bombeEdge {
	e := &bombeEdge{pos: pos, node1: n1, node2: n2}
	n1.edges = append(n1.edges, e)
	n2.edges = append(n2.edges, e)
	return e
}

func (e *bombeEdge) getOther(n *bombeNode) *bombeNode {
	if e.node1 == n {
		return e.node2
	}
	return e.node1
}

// sharedScrambler holds the rotor/reflector state shared by every scrambler (all
// rotors except the fast one), with route caches. -1 marks an unfilled cache.
type sharedScrambler struct {
	reflector   *enigmaReflector
	rotors      []*enigmaRotor
	rotorsRev   []*enigmaRotor
	lowerCache  [26]int
	higherCache [26][26]int
}

func newSharedScrambler(rotors []*enigmaRotor, reflector *enigmaReflector) *sharedScrambler {
	s := &sharedScrambler{}
	s.changeRotors(rotors, reflector)
	return s
}

func (s *sharedScrambler) changeRotors(rotors []*enigmaRotor, reflector *enigmaReflector) {
	s.reflector = reflector
	s.rotors = rotors
	s.rotorsRev = reverseRotors(rotors)
	s.cacheGen()
}

// step advances the first n-1 shared rotors and regenerates the caches.
func (s *sharedScrambler) step(n int) {
	for i := 0; i < n-1; i++ {
		s.rotors[i].step()
	}
	s.cacheGen()
}

// cacheGen pregenerates the route through the shared rotors + reflector + back,
// and clears the just-in-time full-route cache.
func (s *sharedScrambler) cacheGen() {
	for i := range 26 {
		s.lowerCache[i] = -1
		for j := range 26 {
			s.higherCache[i][j] = -1
		}
	}
	for i := range 26 {
		if s.lowerCache[i] != -1 {
			continue
		}
		letter := i
		for _, r := range s.rotors {
			letter = r.transform(letter)
		}
		letter = s.reflector.transform(letter)
		for _, r := range s.rotorsRev {
			letter = r.revTransform(letter)
		}
		s.lowerCache[i] = letter
		s.lowerCache[letter] = i
	}
}

func (s *sharedScrambler) transform(i int) int { return s.lowerCache[i] }

// scrambler is a pseudo-Enigma for one menu edge: the shared state plus its own
// fast rotor.
type scrambler struct {
	base       *sharedScrambler
	initialPos int
	rotor      *enigmaRotor
	end1, end2 int // -1 for the synthetic indicator
	cache      *[26]int
}

func newScrambler(base *sharedScrambler, rotor *enigmaRotor, pos, end1, end2 int) *scrambler {
	sc := &scrambler{base: base, initialPos: pos, end1: end1, end2: end2}
	sc.changeRotor(rotor)
	sc.cache = &base.higherCache[pos]
	return sc
}

func (sc *scrambler) changeRotor(rotor *enigmaRotor) {
	sc.rotor = rotor
	sc.rotor.pos += sc.initialPos
}

func (sc *scrambler) step() {
	sc.rotor.step()
	sc.cache = &sc.base.higherCache[sc.rotor.pos]
}

func (sc *scrambler) transform(i int) int {
	if cached := sc.cache[i]; cached != -1 {
		return cached
	}
	letter := sc.rotor.transform(i)
	letter = sc.base.transform(letter)
	letter = sc.rotor.revTransform(letter)
	sc.cache[i] = letter
	sc.cache[letter] = i
	return letter
}

func (sc *scrambler) getOtherEnd(end int) int {
	if sc.end1 == end {
		return sc.end2
	}
	return sc.end1
}

// getPos reports the rotor positions (left to right), rolling the fast rotor
// back by one to account for Enigma's step-before-encrypt.
func (sc *scrambler) getPos() string {
	b := make([]byte, 0, len(sc.base.rotors)+1)
	b = append(b, byte('A'+mod26(sc.rotor.pos-1, 26))) // #nosec G115 -- letter index in [0,25], so 'A'+n is a valid ASCII byte
	for _, r := range sc.base.rotors {
		b = append(b, byte('A'+r.pos)) // #nosec G115 -- letter index in [0,25], so 'A'+n is a valid ASCII byte
	}
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// bombeMachine is the Bombe simulator.
type bombeMachine struct {
	ciphertext, crib string
	baseRotors       []*enigmaRotor
	check            bool
	nLoops           int
	wires            []bool
	scramblers       [26][]*scrambler
	shared           *sharedScrambler
	allScramblers    []*scrambler
	indicator        *scrambler
	testRegister     int
	testInput        [2]int
	energiseCount    int
}

func newBombeMachine(rotors []string, reflector *enigmaReflector, ciphertext, crib string, check bool) (*bombeMachine, error) {
	if len(ciphertext) < len(crib) {
		return nil, errors.New("Crib overruns supplied ciphertext") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if len(crib) < 2 {
		return nil, errors.New("Crib is too short") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if len(crib) > 25 {
		return nil, errors.New("Crib is too long") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	for i := 0; i < len(crib); i++ {
		if ciphertext[i] == crib[i] {
			return nil, fmt.Errorf("Invalid crib: character %c at pos %d in both ciphertext and crib", ciphertext[i], i) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
	}
	m := &bombeMachine{ciphertext: ciphertext, crib: crib, check: check}
	if err := m.initRotors(rotors); err != nil {
		return nil, err
	}
	mostConnected, edges := m.makeMenu()

	m.wires = make([]bool, 26*26)
	m.shared = newSharedScrambler(sliceRotors(m.baseRotors), reflector)
	for _, edge := range edges {
		end1 := int(edge.node1.letter - 'A')
		end2 := int(edge.node2.letter - 'A')
		sc := newScrambler(m.shared, m.baseRotors[0].copy(), edge.pos, end1, end2)
		if edge.pos == 0 {
			m.indicator = sc
		}
		m.scramblers[end1] = append(m.scramblers[end1], sc)
		m.scramblers[end2] = append(m.scramblers[end2], sc)
		m.allScramblers = append(m.allScramblers, sc)
	}
	if m.indicator == nil {
		m.indicator = newScrambler(m.shared, m.baseRotors[0].copy(), 0, -1, -1)
		m.allScramblers = append(m.allScramblers, m.indicator)
	}
	m.testRegister = int(mostConnected.letter - 'A')
	for _, edge := range mostConnected.edges {
		m.testInput = [2]int{m.testRegister, int(edge.getOther(mostConnected).letter - 'A')}
		break
	}
	return m, nil
}

func (m *bombeMachine) initRotors(rotors []string) error {
	m.baseRotors = nil
	for _, rstr := range rotors {
		r, err := newEnigmaRotor(rstr, "", "A", "A")
		if err != nil {
			return err
		}
		m.baseRotors = append(m.baseRotors, r)
	}
	return nil
}

// changeRotors swaps the rotors/reflector for a subsequent run (Multiple Bombe).
func (m *bombeMachine) changeRotors(rotors []string, reflector *enigmaReflector) error {
	if err := m.initRotors(rotors); err != nil {
		return err
	}
	m.shared.changeRotors(sliceRotors(m.baseRotors), reflector)
	for _, sc := range m.allScramblers {
		sc.changeRotor(m.baseRotors[0].copy())
	}
	return nil
}

// dfsResult is the result of a menu-subgraph depth-first search.
type dfsResult struct {
	loops, nNodes int
	mostConnected *bombeNode
	nConnections  int
	edges         []*bombeEdge
}

func (m *bombeMachine) dfs(node *bombeNode) dfsResult {
	loops := 0
	nNodes := 1
	mostConnected := node
	nConnections := len(node.edges)
	var edges []*bombeEdge
	seen := map[*bombeEdge]bool{}
	add := func(e *bombeEdge) {
		if !seen[e] {
			seen[e] = true
			edges = append(edges, e)
		}
	}
	node.visited = true
	for _, edge := range node.edges {
		if edge.visited {
			continue
		}
		edge.visited = true
		add(edge)
		other := edge.getOther(node)
		if other.visited {
			loops++
			continue
		}
		o := m.dfs(other)
		loops += o.loops
		nNodes += o.nNodes
		for _, e := range o.edges {
			add(e)
		}
		if o.nConnections > nConnections {
			mostConnected = o.mostConnected
			nConnections = o.nConnections
		}
	}
	return dfsResult{loops, nNodes, mostConnected, nConnections, edges}
}

// makeMenu builds the menu graph and returns the most connected node and the
// edges of the subgraph with the most loops (ties broken by node count).
func (m *bombeMachine) makeMenu() (*bombeNode, []*bombeEdge) {
	nodes := map[byte]*bombeNode{}
	var order []byte
	for i := 0; i < len(m.ciphertext); i++ {
		if _, ok := nodes[m.ciphertext[i]]; !ok {
			nodes[m.ciphertext[i]] = &bombeNode{letter: m.ciphertext[i]}
			order = append(order, m.ciphertext[i])
		}
	}
	for i := 0; i < len(m.crib); i++ {
		if _, ok := nodes[m.crib[i]]; !ok {
			nodes[m.crib[i]] = &bombeNode{letter: m.crib[i]}
			order = append(order, m.crib[i])
		}
	}
	for i := 0; i < len(m.crib); i++ {
		newBombeEdge(i, nodes[m.crib[i]], nodes[m.ciphertext[i]])
	}
	var graphs []dfsResult
	for _, c := range order {
		if nodes[c].visited {
			continue
		}
		graphs = append(graphs, m.dfs(nodes[c]))
	}
	sort.SliceStable(graphs, func(i, j int) bool {
		if graphs[i].loops != graphs[j].loops {
			return graphs[i].loops > graphs[j].loops
		}
		return graphs[i].nNodes > graphs[j].nNodes
	})
	m.nLoops = graphs[0].loops
	return graphs[0].mostConnected, graphs[0].edges
}

// energise recursively follows the electrical current from wire (i,j), including
// Welchman's diagonal board (i↔j symmetry) and every attached scrambler.
func (m *bombeMachine) energise(i, j int) {
	idx := 26*i + j
	if m.wires[idx] {
		return
	}
	m.wires[idx] = true
	m.wires[26*j+i] = true
	if i == m.testRegister || j == m.testRegister {
		m.energiseCount++
		if m.energiseCount == 26 {
			return
		}
	}
	for _, sc := range m.scramblers[i] {
		out := sc.transform(j)
		other := sc.getOtherEnd(i)
		if !m.wires[26*other+out] {
			m.energise(other, out)
			if m.energiseCount == 26 {
				return
			}
		}
	}
	if i == j {
		return
	}
	for _, sc := range m.scramblers[j] {
		out := sc.transform(i)
		other := sc.getOtherEnd(j)
		if !m.wires[26*other+out] {
			m.energise(other, out)
			if m.energiseCount == 26 {
				return
			}
		}
	}
}

// tryDecrypt runs a trial decryption at the current setting with the detected
// stecker applied (limited to 26 characters).
func (m *bombeMachine) tryDecrypt(stecker string) string {
	fastRotor := m.indicator.rotor
	initialPos := fastRotor.pos
	plugboard, _ := newEnigmaPairMap(stecker, "Plugboard") // stecker is always a valid pair list
	n := min(26, len(m.ciphertext))
	res := make([]byte, 0, n)
	for i := range n {
		t := m.indicator.transform(plugboard.transform(int(m.ciphertext[i] - 'A')))
		res = append(res, byte('A'+plugboard.transform(t))) // #nosec G115 -- letter index in [0,25], so 'A'+n is a valid ASCII byte
		m.indicator.step()
	}
	fastRotor.pos = initialPos
	return string(res)
}

func bombeFormatPair(a, b int) string {
	if a > b {
		a, b = b, a
	}
	return string([]byte{byte('A' + a), byte('A' + b)}) // #nosec G115 -- letter index in [0,25], so 'A'+n is a valid ASCII byte
}

// checkingMachine verifies a stop by walking the menu from the hypothesised
// stecker pair, returning the discovered plugboard string or "" if contradictory.
func (m *bombeMachine) checkingMachine(pair int) string {
	if pair != m.testInput[1] {
		for i := range m.wires {
			m.wires[i] = false
		}
		m.energiseCount = 0
		m.energise(m.testRegister, pair)
	}
	var results []string
	seen := map[string]bool{}
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			results = append(results, s)
		}
	}
	add(bombeFormatPair(m.testRegister, pair))
	for i := range 26 {
		count := 0
		other := 0
		for j := range 26 {
			if m.wires[i*26+j] {
				count++
				other = j
			}
		}
		if count > 1 {
			return ""
		}
		if count == 0 {
			continue
		}
		add(bombeFormatPair(i, other))
	}
	return strings.Join(results, " ")
}

// checkStop processes a potential stop, returning [rotor pos, plugboard,
// decryption preview] or ok=false for no stop.
func (m *bombeMachine) checkStop() ([]string, bool) {
	count := m.energiseCount
	if count == 26 {
		return nil, false
	}
	var steckerPair int
	switch count {
	case 25:
		for j := range 26 {
			if !m.wires[26*m.testRegister+j] {
				steckerPair = j
				break
			}
		}
	case 1:
		steckerPair = m.testInput[1]
	default:
		// "Boxing stop" - a stop but not a single hypothesis.
		if !m.check {
			return []string{m.indicator.getPos(), "??", m.tryDecrypt("")}, true
		}
		stecker := ""
		found := false
		for i := range 26 {
			newStecker := m.checkingMachine(i)
			if newStecker != "" {
				if found {
					return []string{m.indicator.getPos(), "??", m.tryDecrypt("")}, true
				}
				stecker = newStecker
				found = true
			}
		}
		if !found {
			return nil, false
		}
		return []string{m.indicator.getPos(), stecker, m.tryDecrypt(stecker)}, true
	}
	var stecker string
	if m.check {
		stecker = m.checkingMachine(steckerPair)
		if stecker == "" {
			return nil, false
		}
	} else {
		stecker = string([]byte{byte('A' + m.testRegister), byte('A' + steckerPair)}) // #nosec G115 -- letter index in [0,25], so 'A'+n is a valid ASCII byte
	}
	return []string{m.indicator.getPos(), stecker, m.tryDecrypt(stecker)}, true
}

// run tries every rotor setting and collects the surviving candidate stops.
func (m *bombeMachine) run() [][]string {
	var result [][]string
	nChecks := 1
	for range m.baseRotors {
		nChecks *= 26
	}
	for i := 1; i <= nChecks; i++ {
		for k := range m.wires {
			m.wires[k] = false
		}
		m.energiseCount = 0
		m.energise(m.testInput[0], m.testInput[1])
		if stop, ok := m.checkStop(); ok {
			result = append(result, stop)
		}
		// Count how many rotors have returned to their start (so the next steps).
		n := 1
		for j := 1; j < len(m.baseRotors); j++ {
			if i%powInt(26, j) == 0 {
				n++
			} else {
				break
			}
		}
		if n > 1 {
			m.shared.step(n)
		}
		for _, sc := range m.allScramblers {
			sc.step()
		}
	}
	return result
}

// --- helpers ---

// sliceRotors returns the shared rotors (everything but the fast/bottom rotor).
func sliceRotors(rotors []*enigmaRotor) []*enigmaRotor { return rotors[1:] }

func reverseRotors(rotors []*enigmaRotor) []*enigmaRotor {
	rev := make([]*enigmaRotor, len(rotors))
	for i, r := range rotors {
		rev[len(rotors)-1-i] = r
	}
	return rev
}

func reverseStrings(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func powInt(base, exp int) int {
	res := 1
	for range exp {
		res *= base
	}
	return res
}

// sliceFrom mirrors JS String.slice(offset): an offset past the end yields "".
func sliceFrom(s string, offset int) string {
	if offset >= len(s) {
		return ""
	}
	return s[offset:]
}

// bombeStripStepping drops the "<steps" suffix from a rotor spec.
func bombeStripStepping(rstr string) string {
	if strings.Contains(rstr, "<") {
		return strings.Split(rstr, "<")[0]
	}
	return rstr
}

// bombeTableHeader is the shared HTML table opening.
const bombeTableHeader = "<table class='table table-hover table-sm table-bordered table-nonfluid'><tr><th>Rotor stops</th>  <th>Partial plugboard</th>  <th>Decryption preview</th></tr>\n"

func bombeLoopWord(n int) string {
	if n == 1 {
		return "loop"
	}
	return "loops"
}

func bombeRows(b *strings.Builder, result [][]string) {
	for _, r := range result {
		fmt.Fprintf(b, "<tr><td>%s</td>  <td>%s</td>  <td>%s</td></tr>\n", r[0], r[1], r[2])
	}
}

// --- Bombe operation ---

// Bombe attacks Enigma with a single known rotor configuration.
type Bombe struct{}

// Meta returns the operation metadata.
func (Bombe) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Bombe",
		Module:      "Bletchley",
		Description: "Emulation of the Bombe machine used at Bletchley Park to attack Enigma, based on work by Polish and British cryptanalysts.<br><br>To run this you need to have a 'crib', which is some known plaintext for a chunk of the target ciphertext, and know the rotors used. (See the 'Bombe (multiple runs)' operation if you don't know the rotors.) The machine will suggest possible configurations of the Enigma. Each suggestion has the rotor start positions (left to right) and known plugboard pairs.<br><br>Choosing a crib: First, note that Enigma cannot encrypt a letter to itself, which allows you to rule out some positions for possible cribs. Secondly, the Bombe does not simulate the Enigma's middle rotor stepping. The longer your crib, the more likely a step happened within it, which will prevent the attack working. However, other than that, longer cribs are generally better. The attack produces a 'menu' which maps ciphertext letters to plaintext, and the goal is to produce 'loops': for example, with ciphertext ABC and crib CAB, we have the mappings A&lt;-&gt;C, B&lt;-&gt;A, and C&lt;-&gt;B, which produces a loop A-B-C-A. The more loops, the better the crib. The operation will output this: if your menu has too few loops or is too short, a large number of incorrect outputs will usually be produced. Try a different crib. If the menu seems good but the right answer isn't produced, your crib may be wrong, or you may have overlapped the middle rotor stepping - try a different crib.<br><br>Output is not sufficient to fully decrypt the data. You will have to recover the rest of the plugboard settings by inspection. And the ring position is not taken into account: this affects when the middle rotor steps. If your output is correct for a bit, and then goes wrong, adjust the ring and start position on the right-hand rotor together until the output improves. If necessary, repeat for the middle rotor.<br><br>By default this operation runs the checking machine, a manual process to verify the quality of Bombe stops, on each stop, discarding stops which fail. If you want to see how many times the hardware actually stops for a given input, disable the checking machine.<br><br>More detailed descriptions of the Enigma, Typex and Bombe operations <a href='https://github.com/gchq/CyberChef/wiki/Enigma,-the-Bombe,-and-Typex'>can be found here</a>.",
		InfoURL:     "https://wikipedia.org/wiki/Bombe",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Bombe) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Model", Type: core.ArgOption, Value: []string{"3-rotor", "4-rotor"}},
		{Name: "Left-most (4th) rotor", Type: core.ArgEditableOption, Value: enigmaRotorsFourth[0].value},
		{Name: "Left-hand rotor", Type: core.ArgEditableOption, Value: enigmaRotorDefs[0].value},
		{Name: "Middle rotor", Type: core.ArgEditableOption, Value: enigmaRotorDefs[1].value},
		{Name: "Right-hand rotor", Type: core.ArgEditableOption, Value: enigmaRotorDefs[2].value},
		{Name: "Reflector", Type: core.ArgEditableOption, Value: enigmaReflectorDefs[0].value},
		{Name: "Crib", Type: core.ArgString, Value: ""},
		{Name: "Crib offset", Type: core.ArgNumber, Value: 0},
		{Name: "Use checking machine", Type: core.ArgBoolean, Value: true},
	}
}

// Run performs the attack. Ported from CyberChef Bombe.mjs (+ present()).
func (Bombe) Run(in *core.Dish, args []any) (*core.Dish, error) {
	model := args[0].(string)
	crib := args[6].(string)
	offset := int(args[7].(float64))
	check := args[8].(bool)

	var rotors []string
	for i := range 4 {
		if i == 0 && model == "3-rotor" {
			continue
		}
		rotors = append(rotors, bombeStripStepping(args[i+1].(string)))
	}
	reverseStrings(rotors)

	if len(crib) == 0 {
		return nil, errors.New("Crib cannot be empty") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if offset < 0 {
		return nil, errors.New("Offset cannot be negative") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	input := strings.ToUpper(enigmaNonAlpha.ReplaceAllString(in.String(), ""))
	crib = strings.ToUpper(enigmaNonAlpha.ReplaceAllString(crib, ""))
	ciphertext := sliceFrom(input, offset)

	reflector, err := newEnigmaReflector(args[5].(string))
	if err != nil {
		return nil, err
	}
	bombe, err := newBombeMachine(rotors, reflector, ciphertext, crib, check)
	if err != nil {
		return nil, err
	}
	result := bombe.run()

	var b strings.Builder
	fmt.Fprintf(&b, "Bombe run on menu with %d %s (2+ desirable). Note: Rotor positions are listed left to right and start at the beginning of the crib, and ignore stepping and the ring setting. Some plugboard settings are determined. A decryption preview starting at the beginning of the crib and ignoring stepping is also provided.\n\n", bombe.nLoops, bombeLoopWord(bombe.nLoops))
	b.WriteString(bombeTableHeader)
	bombeRows(&b, result)
	b.WriteString("</table>")
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}

// --- Multiple Bombe operation ---

// multiBombeRun is one rotor-configuration run with results.
type multiBombeRun struct {
	rotors    []string
	reflector string
	result    [][]string
}

// MultipleBombe runs the Bombe over many rotor configurations.
type MultipleBombe struct{}

// Meta returns the operation metadata.
func (MultipleBombe) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Multiple Bombe",
		Module:      "Bletchley",
		Description: "Emulation of the Bombe machine used to attack Enigma. This version carries out multiple Bombe runs to handle unknown rotor configurations.<br><br>You should test your menu on the single Bombe operation before running it here. See the description of the Bombe operation for instructions on choosing a crib.<br><br>More detailed descriptions of the Enigma, Typex and Bombe operations <a href='https://github.com/gchq/CyberChef/wiki/Enigma,-the-Bombe,-and-Typex'>can be found here</a>.",
		InfoURL:     "https://wikipedia.org/wiki/Bombe",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (MultipleBombe) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Standard Enigmas", Type: core.ArgOption, Value: []string{
			"German Service Enigma (First - 3 rotor)",
			"German Service Enigma (Second - 3 rotor)",
			"German Service Enigma (Third - 4 rotor)",
			"German Service Enigma (Fourth - 4 rotor)",
			"User defined",
		}},
		{Name: "Main rotors", Type: core.ArgString, Value: ""},
		{Name: "4th rotor", Type: core.ArgString, Value: ""},
		{Name: "Reflectors", Type: core.ArgString, Value: ""},
		{Name: "Crib", Type: core.ArgString, Value: ""},
		{Name: "Crib offset", Type: core.ArgNumber, Value: 0},
		{Name: "Use checking machine", Type: core.ArgBoolean, Value: true},
	}
}

// multiBombeValidateRotor strips stepping and validates a rotor wiring string.
func multiBombeValidateRotor(rstr string) (string, error) {
	rstr = bombeStripStepping(rstr)
	if !enigmaWiringRe.MatchString(rstr) {
		return "", errors.New("Rotor wiring must be 26 unique uppercase letters") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	var seen [26]bool
	uniq := 0
	for i := range 26 {
		if b := rstr[i] - 'A'; !seen[b] {
			seen[b] = true
			uniq++
		}
	}
	if uniq != 26 {
		return "", errors.New("Rotor wiring must be 26 unique uppercase letters") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return rstr, nil
}

// Run performs the multi-configuration attack. Ported from CyberChef
// MultipleBombe.mjs (+ present()).
func (MultipleBombe) Run(in *core.Dish, args []any) (*core.Dish, error) {
	crib := args[4].(string)
	offset := int(args[5].(float64))
	check := args[6].(bool)

	var rotors []string
	for rstr := range strings.SplitSeq(args[1].(string), "\n") {
		v, err := multiBombeValidateRotor(rstr)
		if err != nil {
			return nil, err
		}
		rotors = append(rotors, v)
	}
	if len(rotors) < 3 {
		return nil, errors.New("A minimum of three rotors must be supplied") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	var fourthRotors []string
	if fourthStr := args[2].(string); fourthStr != "" {
		for rstr := range strings.SplitSeq(fourthStr, "\n") {
			v, err := multiBombeValidateRotor(rstr)
			if err != nil {
				return nil, err
			}
			fourthRotors = append(fourthRotors, v)
		}
	}
	if len(fourthRotors) == 0 {
		fourthRotors = append(fourthRotors, "")
	}
	var reflectors []*enigmaReflector
	for rstr := range strings.SplitSeq(args[3].(string), "\n") {
		refl, err := newEnigmaReflector(rstr)
		if err != nil {
			return nil, err
		}
		reflectors = append(reflectors, refl)
	}
	if len(crib) == 0 {
		return nil, errors.New("Crib cannot be empty") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	if offset < 0 {
		return nil, errors.New("Offset cannot be negative") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	input := strings.ToUpper(enigmaNonAlpha.ReplaceAllString(in.String(), ""))
	crib = strings.ToUpper(enigmaNonAlpha.ReplaceAllString(crib, ""))
	ciphertext := sliceFrom(input, offset)

	var bombe *bombeMachine
	nLoops := 0
	var runs []multiBombeRun
	for _, rotor1 := range rotors {
		for _, rotor2 := range rotors {
			if rotor2 == rotor1 {
				continue
			}
			for _, rotor3 := range rotors {
				if rotor3 == rotor2 || rotor3 == rotor1 {
					continue
				}
				for _, rotor4 := range fourthRotors {
					for _, reflector := range reflectors {
						runRotors := []string{rotor1, rotor2, rotor3}
						if rotor4 != "" {
							runRotors = append(runRotors, rotor4)
						}
						if bombe == nil {
							var err error
							bombe, err = newBombeMachine(runRotors, reflector, ciphertext, crib, check)
							if err != nil {
								return nil, err
							}
							nLoops = bombe.nLoops
						} else if err := bombe.changeRotors(runRotors, reflector); err != nil {
							return nil, err
						}
						result := bombe.run()
						if len(result) > 0 {
							runs = append(runs, multiBombeRun{rotors: runRotors, reflector: reflector.pairs, result: result})
						}
					}
				}
			}
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Bombe run on menu with %d %s (2+ desirable). Note: Rotors and rotor positions are listed left to right, ignore stepping and the ring setting, and positions start at the beginning of the crib. Some plugboard settings are determined. A decryption preview starting at the beginning of the crib and ignoring stepping is also provided.\n", nLoops, bombeLoopWord(nLoops))
	for _, run := range runs {
		rev := append([]string{}, run.rotors...)
		reverseStrings(rev)
		fmt.Fprintf(&b, "\nRotors: %s\nReflector: %s\n", strings.Join(rev, ", "), run.reflector)
		b.WriteString(bombeTableHeader)
		bombeRows(&b, run.result)
		b.WriteString("</table>\n")
	}
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}
