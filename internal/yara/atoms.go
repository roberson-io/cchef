package yara

import "strings"

// Atom quality, which is how libyara decides a string is too weak to search for
// efficiently and says so.
//
// Every string is reduced to a short run of bytes — an atom — that a scan can
// look for quickly. A string whose best atom scores poorly makes the scan slow,
// because the atom turns up everywhere and each hit has to be checked in full.
// The scoring is libyara's own, from atoms.c.

const (
	// atomLength is the most bytes an atom holds.
	atomLength = 4
	// maxQuality is the best score there is, before the offset below.
	maxQuality = 255
	// qualityOffset lifts the score so that the best possible atom reaches
	// maxQuality: twenty points a byte and two more for each distinct one.
	qualityOffset = maxQuality - 22*atomLength
	// warnBelow is the raw score under which libyara says a string will slow
	// the scan down.
	warnBelow = 38

	// What a byte is worth. The commonest bytes say least about where they are,
	// and a letter says a little less than anything else because matching it
	// without regard to case doubles the atoms it stands for.
	commonByteScore = 12
	letterScore     = 18
	otherByteScore  = 20
	// uniqueByteBonus rewards an atom whose bytes differ from one another.
	uniqueByteBonus = 2
	// runPenalty is charged per byte against an atom that is one common byte
	// over and over, which is the worst thing to search for.
	runPenalty = 10
)

// commonBytes are the ones that turn up everywhere in ordinary data.
var commonBytes = map[byte]bool{0x00: true, 0x20: true, 0xCC: true, 0xFF: true}

// runBytes are the ones a long run of which says almost nothing.
var runBytes = map[byte]bool{0x00: true, 0x20: true, 0x90: true, 0xCC: true, 0xFF: true}

// atomQuality scores one atom. The result is not zero-based: it counts down
// from the best possible atom.
func atomQuality(atom []byte) int {
	return qualityOffset + rawAtomQuality(atom)
}

// rawAtomQuality is the score before it is lifted, which is what the warning
// threshold is written against.
func rawAtomQuality(atom []byte) int {
	quality := 0
	seen := map[byte]bool{}
	for _, b := range atom {
		switch {
		case commonBytes[b]:
			quality += commonByteScore
		case isLetter(b):
			quality += letterScore
		default:
			quality += otherByteScore
		}
		seen[b] = true
	}

	// One byte over and over is penalised heavily when that byte is a common
	// one; otherwise the variety itself is worth something.
	if len(seen) == 1 && onlyRunByte(seen) {
		return quality - runPenalty*len(atom)
	}
	return quality + uniqueByteBonus*len(seen)
}

// isLetter reports whether a byte is a letter in either case.
func isLetter(b byte) bool { return lower(b) >= 'a' && lower(b) <= 'z' }

// onlyRunByte reports whether the single byte an atom is made of is one of the
// ones a run of says nothing.
func onlyRunByte(seen map[byte]bool) bool {
	for b := range seen {
		if !runBytes[b] {
			return false
		}
	}
	return true
}

// bestAtom picks the strongest run of bytes a literal offers, which is the one
// a scan would search for.
func bestAtom(literal []byte) []byte {
	if len(literal) <= atomLength {
		return literal
	}
	best := literal[:atomLength]
	bestScore := rawAtomQuality(best)
	for i := 1; i+atomLength <= len(literal); i++ {
		window := literal[i : i+atomLength]
		if score := rawAtomQuality(window); score > bestScore {
			best, bestScore = window, score
		}
	}
	return best
}

// slowsScanning reports whether libyara would warn that a string will slow a
// scan down. What the scan looks for is not always the string as written: wide
// searches for the characters spread out with zeroes between them, and xor
// searches for the atom under every key in its range. libyara scores the string
// as written and then every form the modifiers turn it into, and keeps the
// weakest of them.
func slowsScanning(str *String) bool {
	atom := bestAtom(literalRun(str))
	quality := rawAtomQuality(atom)
	for _, form := range modifiedAtoms(atom, str.Mods) {
		quality = min(quality, rawAtomQuality(form))
	}
	return quality < warnBelow
}

// modifiedAtoms is every form of an atom a scan would look for once the
// modifiers are applied. Matching without regard to case is left out on
// purpose: a letter scores the same in either case and can only add to the
// variety, so no case of an atom is ever weaker than the atom itself.
func modifiedAtoms(atom []byte, mods Modifiers) [][]byte {
	atoms := [][]byte{atom}
	if mods.Wide {
		wide := widenAtoms(atoms)
		if mods.ASCII {
			atoms = append(atoms, wide...)
		} else {
			atoms = wide
		}
	}
	if mods.XOR {
		atoms = xorAtoms(atoms, mods.XORMin, mods.XORMax)
	}
	return atoms
}

// widenAtoms spreads each atom's bytes out with a zero after each, which is
// what a scan looks for when a string is written as wide characters.
func widenAtoms(atoms [][]byte) [][]byte {
	out := make([][]byte, 0, len(atoms))
	for _, atom := range atoms {
		wide := make([]byte, 0, atomLength)
		for _, b := range atom {
			if len(wide) == atomLength {
				break
			}
			wide = append(wide, b, 0)
		}
		out = append(out, wide)
	}
	return out
}

// xorAtoms is each atom under every key it would be searched for with.
func xorAtoms(atoms [][]byte, low, high int) [][]byte {
	out := make([][]byte, 0, len(atoms)*(high-low+1))
	for _, atom := range atoms {
		for key := low; key <= high; key++ {
			keyed := make([]byte, len(atom))
			for i, b := range atom {
				keyed[i] = b ^ byte(key) // #nosec G115 -- a key is one byte
			}
			out = append(out, keyed)
		}
	}
	return out
}

// literalRun finds the longest stretch of fixed bytes a string offers, which is
// what a scan would search for. Plain text is fixed throughout; a hex pattern
// or a regular expression is fixed only in places.
func literalRun(str *String) []byte {
	if str.Kind == stringText {
		return []byte(str.Text)
	}
	if str.Kind == stringHex {
		return longestRun(hexFixedRuns(str.Text))
	}
	return longestRun(regexLiteralRuns(str.Text))
}

// longestRun picks the longest of several stretches.
func longestRun(runs [][]byte) []byte {
	var best []byte
	for _, run := range runs {
		if len(run) > len(best) {
			best = run
		}
	}
	return best
}

// hexFixedRuns picks out the stretches of a hex pattern that name exact bytes,
// broken wherever a wildcard, a jump or a choice makes the next byte unknown.
func hexFixedRuns(body string) [][]byte {
	var runs [][]byte
	var run []byte
	for i := 0; i < len(body); {
		switch c := body[i]; {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case isHexDigit(c) && i+1 < len(body) && isHexDigit(body[i+1]):
			run = append(run, byte(hexValue(c)<<4|hexValue(body[i+1]))) // #nosec G115 -- two hex digits
			i += 2
		default:
			runs, run = append(runs, run), nil
			i++
		}
	}
	return append(runs, run)
}

// regexLiteralRuns picks out the stretches of a regular expression that are
// plain characters. A class, or an escape that stands for a whole class of
// bytes, offers nothing to look for; a count writes its character out as many
// times as it must appear; and a quantifier that allows none of the character
// before it takes that character away again.
func regexLiteralRuns(body string) [][]byte {
	var runs [][]byte
	var run []byte
	end := func() { runs, run = append(runs, run), nil }

	for i := 0; i < len(body); {
		switch c := body[i]; {
		case c == '\\':
			b, next, literal := regexEscape(body, i)
			if literal {
				run = append(run, b)
			} else {
				end()
			}
			i = next
		case c == '[':
			end()
			i = skipClass(body, i)
		case c == '{':
			i = readRepeat(body, i, &run, end)
		case c == '*' || c == '?':
			run, i = repeatRun(run, 0), i+1
			end()
		case c == '(' || c == ')':
			// Brackets only group, so the letters either side of them are still
			// one run. What follows a group and says how many times it repeats
			// ends the run in its own right, as it does anywhere else.
			i++
		case strings.ContainsRune(`.|+^$`, rune(c)):
			end()
			i++
		default:
			run, i = append(run, c), i+1
		}
	}
	return append(runs, run)
}

// readRepeat applies a count in braces to the run built up so far and reports
// where the expression carries on. A brace that does not open a count stands
// for itself, so it joins the run like any other plain character.
func readRepeat(body string, i int, run *[]byte, end func()) int {
	least, atMost, next, isCount := repeatCount(body, i)
	if !isCount {
		*run = append(*run, body[i])
		return i + 1
	}
	*run = repeatRun(*run, least)
	// Only a count that is exactly met leaves the run whole: anything beyond
	// the fewest is optional, so what follows is not joined onto it.
	if !atMost || least == 0 {
		end()
	}
	return next
}

// repeatRun writes the last character of a run out as many times as a count
// says it must appear, or drops it when the count allows none at all.
func repeatRun(run []byte, times int) []byte {
	if len(run) == 0 {
		return run
	}
	last := run[len(run)-1]
	if times == 0 {
		return run[:len(run)-1]
	}
	for i := 1; i < times; i++ {
		run = append(run, last)
	}
	return run
}

// repeatCount reads a count in braces: the fewest times the character before it
// must appear, and whether that is also the most. A range may leave the first
// number out, in which case the character need not appear at all.
func repeatCount(body string, i int) (least int, atMost bool, next int, isCount bool) {
	at := i + 1
	digits := at
	for at < len(body) && body[at] >= '0' && body[at] <= '9' {
		least = least*10 + int(body[at]-'0')
		at++
	}

	atMost = true
	switch {
	case at < len(body) && body[at] == ',':
		atMost = false
		for at++; at < len(body) && body[at] >= '0' && body[at] <= '9'; at++ {
		}
	case at == digits:
		return 0, false, i, false
	}
	if at == len(body) || body[at] != '}' {
		return 0, false, i, false
	}
	return least, atMost, at + 1, true
}

// skipClass steps over a character class. A closing bracket straight after the
// opening one, or after a negating caret, stands for itself rather than ending
// the class.
func skipClass(body string, i int) int {
	at := i + 1
	if at < len(body) && body[at] == '^' {
		at++
	}
	if at < len(body) && body[at] == ']' {
		at++
	}
	for at < len(body) {
		switch body[at] {
		case '\\':
			at += 2
		case ']':
			return at + 1
		default:
			at++
		}
	}
	return at
}

// regexEscape reads what a backslash introduces: a byte the run can keep, or a
// class of bytes that it cannot.
func regexEscape(body string, i int) (b byte, next int, literal bool) {
	if i+1 == len(body) {
		return 0, i + 1, false
	}
	switch c := body[i+1]; c {
	case 'x':
		if i+3 < len(body) && isHexDigit(body[i+2]) && isHexDigit(body[i+3]) {
			return byte(hexValue(body[i+2])<<4 | hexValue(body[i+3])), i + 4, true // #nosec G115 -- two hex digits
		}
		return 0, i + 2, false
	case 'n':
		return '\n', i + 2, true
	case 't':
		return '\t', i + 2, true
	case 'r':
		return '\r', i + 2, true
	case 'd', 'D', 'w', 'W', 's', 'S', 'b', 'B':
		return 0, i + 2, false
	default:
		return c, i + 2, true
	}
}

// isHexDigit reports whether a character is one of the sixteen.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (lower(c) >= 'a' && lower(c) <= 'f')
}

// unboundedDot says whether a pattern repeats "anything at all" without an
// upper bound — `.*`, `.+` or `.{x,}` — which makes a scan slow and which YARA
// warns about.
//
// Brackets merely group, so what is inside them still counts. A choice anywhere
// leaves the warning unsaid, since the pattern is then read as a choice rather
// than as a run of pieces.
func unboundedDot(text string) bool {
	found := false
	for i := 0; i < len(text); {
		switch text[i] {
		case '\\':
			i += 2
		case '[':
			i = skipClass(text, i)
		case '|':
			return false
		case '(':
			i++
		case '.':
			var repeats bool
			i, repeats = afterDot(text, i+1)
			found = found || repeats
		default:
			i++
		}
	}
	return found
}

// afterDot reads what follows a full stop and says whether it repeats without an
// upper bound.
func afterDot(text string, i int) (int, bool) {
	if i >= len(text) {
		return i, false
	}
	switch text[i] {
	case '*', '+':
		return i + 1, true
	case '{':
		end := i
		for end < len(text) && text[end] != '}' {
			end++
		}
		if end == len(text) {
			return i + 1, false
		}
		// A count with nothing after its comma has no upper bound.
		count := text[i+1 : end]
		comma := strings.IndexByte(count, ',')
		return end + 1, comma >= 0 && comma == len(count)-1
	}
	return i, false
}
