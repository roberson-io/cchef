package ops

import (
	"regexp"
	"testing"
)

// Rotor/reflector spec strings shared by the fixtures.
const (
	bombeRotorI   = "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R"
	bombeRotorII  = "AJDKSIRUXBLHWTMCQGZNPYFVOE<F"
	bombeRotorIII = "BDFHJLCPRTXVZNYEIWGAKMUSQO<W"
	bombeReflB    = "AY BR CU DH EQ FS GL IP JX KN MO TZ VW"
)

// bombeMatch runs a Bombe-family op and asserts the output (or error message)
// matches the fixture's expectedMatch regex.
func bombeMatch(t *testing.T, op, input, pattern string, args ...any) {
	t.Helper()
	re := regexp.MustCompile(pattern)
	out, err := runOp(t, op, input, args...)
	got := out
	if err != nil {
		got = err.Error()
	}
	if !re.MatchString(got) {
		t.Fatalf("%s output does not match %q\ngot: %s", op, pattern, got)
	}
}

// Cases transcribed from ../CyberChef/tests/operations/tests/Bombe.mjs.
func TestBombeFixtures(t *testing.T) {
	// args: model, 4th rotor, left, middle, right, reflector, crib, offset, check
	t.Run("3 rotor (self-stecker)", func(t *testing.T) {
		bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS",
			`<td>LGA</td> {2}<td>SS</td> {2}<td>VFISUSGTKSTMPSUNAK</td>`,
			"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "THISISATESTMESSAGE", 0, false)
	})
	t.Run("3 rotor (other stecker)", func(t *testing.T) {
		bombeMatch(t, "Bombe", "JBYALIHDYNUAAVKBYM",
			`<td>LGA</td> {2}<td>AG</td> {2}<td>QFIMUMAFKMQSKMYNGW</td>`,
			"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "THISISATESTMESSAGE", 0, false)
	})
	t.Run("crib offset", func(t *testing.T) {
		bombeMatch(t, "Bombe", "AAABBYFLTHHYIJQAYBBYS",
			`<td>LGA</td> {2}<td>SS</td> {2}<td>VFISUSGTKSTMPSUNAK</td>`,
			"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "THISISATESTMESSAGE", 3, false)
	})
	t.Run("multiple stops", func(t *testing.T) {
		bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS",
			`<td>LGA</td> {2}<td>TT</td> {2}<td>VFISUSGTKSTMPSUNAK</td>`,
			"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "THISISATESTM", 0, false)
	})
	t.Run("checking machine", func(t *testing.T) {
		bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS",
			`<td>LGA</td> {2}<td>TT AG BO CL EK FF HH II JJ SS YY</td> {2}<td>THISISATESTMESSAGE</td>`,
			"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "THISISATESTM", 0, true)
	})
}

func TestBombeErrors(t *testing.T) {
	base := func(crib string, offset int) []any {
		return []any{"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, crib, offset, false}
	}
	bombeMatch(t, "Bombe", "JBYALIHDYNUAAVKBYM", `Crib cannot be empty`, base("", 0)...)
	bombeMatch(t, "Bombe", "JBYALIHDYNUAAVKBYM", `Crib is too short`, base("A", 0)...)
	bombeMatch(t, "Bombe", "JBYALIHDYNUAAVKBYM", `Invalid crib: .* in both ciphertext and crib`, base("AAAAAAAA", 0)...)
	bombeMatch(t, "Bombe", "JBYALIHDYNUAAVKBYM", `Crib overruns supplied ciphertext`, base("CCCCCCCCCCCCCCCCCCCCCC", 0)...)
	bombeMatch(t, "Bombe", "BBBBBBBBBBBBBBBBBBBBBBBBBB", `Crib is too long`, base("AAAAAAAAAAAAAAAAAAAAAAAAAA", 0)...)
	bombeMatch(t, "Bombe", "AAAAA", `Offset cannot be negative`, base("BBBBB", -1)...)
}

// Transcribed from ../CyberChef/tests/operations/tests/MultipleBombe.mjs.
func TestMultipleBombeFixtures(t *testing.T) {
	// args: standard enigmas, main rotors, 4th rotor, reflectors, crib, offset, check
	t.Run("3 rotor", func(t *testing.T) {
		bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS",
			`<td>LGA</td> {2}<td>SS</td> {2}<td>VFISUSGTKSTMPSUNAK</td>`,
			"User defined",
			bombeRotorI+"\n"+bombeRotorII+"\n"+bombeRotorIII,
			"", bombeReflB, "THISISATESTMESSAGE", 0, false)
	})
}

// TestBombeAttack runs a full 3-rotor attack (both with and without the checking
// machine), exercising the count==1, count==25 and boxing-stop code paths. The
// correct rotor position (AAA) recovers the full plaintext.
func TestBombeAttack(t *testing.T) {
	// "ATTACKATDAWNXWEATTACK" enciphered from AAA with rotors I/II/III, reflector B.
	const ct = "BZHGNOCRRTCMMUSCEIPIS"
	bombeMatch(t, "Bombe", ct,
		`<td>AAA</td> {2}<td>[A-Z ]*</td> {2}<td>ATTACKATDAWNXWEATTACK</td>`,
		"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "ATTACKATDAWN", 0, true)
	// Without the checking machine every hardware stop is reported (many "??").
	bombeMatch(t, "Bombe", ct, `menu with 0 loops`,
		"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "ATTACKATDAWN", 0, false)
	// A short (weak) crib with the checking machine exercises boxing stops that
	// the checking machine resolves; the full result set matches CyberChef.
	bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS",
		`<td>AAA</td> {2}<td>NT AO BY ES FW HM</td> {2}<td>THLSUSATWUEFPSCQLZ</td>`,
		"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "THISISAT", 0, true)
}

// TestBombe4Rotor covers the 4-rotor path (slow; the equivalent CyberChef fixture
// is disabled for the same reason).
func TestBombe4Rotor(t *testing.T) {
	bombeMatch(t, "Bombe", "LUOXGJSHGEDSRDOQQX",
		`<td>LHSC</td> {2}<td>SS</td> {2}<td>HHHSSSGQUUQPKSEKWK</td>`,
		"4-rotor", "LEYJVCNIXWPBQMDRTAKZGFUHOS", bombeRotorI, bombeRotorII, bombeRotorIII,
		"AE BN CK DQ FU GY HW IJ LO MP RX SZ TV", "THISISATESTMESSAGE", 0, false)
}

// TestBombeMisc covers rotors without stepping markers, an offset past the end,
// and an invalid rotor wiring.
func TestBombeMisc(t *testing.T) {
	// Rotors without a "<steps" suffix (stripping is a no-op) give the same result.
	bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS",
		`<td>LGA</td> {2}<td>SS</td> {2}<td>VFISUSGTKSTMPSUNAK</td>`,
		"3-rotor", "", "EKMFLGDQVZNTOWYHXUSPAIBRCJ", "AJDKSIRUXBLHWTMCQGZNPYFVOE", "BDFHJLCPRTXVZNYEIWGAKMUSQO",
		bombeReflB, "THISISATESTMESSAGE", 0, false)
	// Offset past the end leaves an empty ciphertext (crib then overruns).
	bombeMatch(t, "Bombe", "AAAAA", `Crib overruns supplied ciphertext`,
		"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, bombeReflB, "BB", 99, false)
	// An invalid rotor wiring is rejected.
	bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS", `Rotor wiring must be 26 unique uppercase letters`,
		"3-rotor", "", "SHORT", bombeRotorII, bombeRotorIII, bombeReflB, "THISISATESTMESSAGE", 0, false)
	// An invalid reflector is rejected.
	bombeMatch(t, "Bombe", "BBYFLTHHYIJQAYBBYS", `Reflector must have exactly 13 pairs covering every letter`,
		"3-rotor", "", bombeRotorI, bombeRotorII, bombeRotorIII, "AB", "THISISATESTMESSAGE", 0, false)
}

// TestMultipleBombeErrors covers Multiple Bombe's validation paths.
func TestMultipleBombeErrors(t *testing.T) {
	mb := func(main, fourth, refl, crib string) []any {
		return []any{"User defined", main, fourth, refl, crib, 0, false}
	}
	valid3 := bombeRotorI + "\n" + bombeRotorII + "\n" + bombeRotorIII
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `Rotor wiring must be 26 unique uppercase letters`,
		mb("SHORT\n"+bombeRotorII+"\n"+bombeRotorIII, "", bombeReflB, "THISISATESTMESSAGE")...)
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `A minimum of three rotors must be supplied`,
		mb(bombeRotorI+"\n"+bombeRotorII, "", bombeReflB, "THISISATESTMESSAGE")...)
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `Rotor wiring must be 26 unique uppercase letters`,
		mb(valid3, "BADFOURTH", bombeReflB, "THISISATESTMESSAGE")...)
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `Reflector must have exactly 13 pairs covering every letter`,
		mb(valid3, "", "AB", "THISISATESTMESSAGE")...)
	// A 26-character rotor with a repeated letter passes the length regex but
	// fails the uniqueness check.
	dupRotor := "EKMFLGDQVZNTOWYHXUSPAIBRCA" // final letter A repeats
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `Rotor wiring must be 26 unique uppercase letters`,
		mb(dupRotor+"\n"+bombeRotorII+"\n"+bombeRotorIII, "", bombeReflB, "THISISATESTMESSAGE")...)
	// A valid 4th rotor is added to the run; an overrunning crib then fails the
	// first run before any (slow) 4-rotor attack executes.
	bombeMatch(t, "Multiple Bombe", "BBY", `Crib overruns supplied ciphertext`,
		mb(valid3, "LEYJVCNIXWPBQMDRTAKZGFUHOS", bombeReflB, "THISISATESTMESSAGE")...)
	// Non-empty crib that overruns the ciphertext fails in the first Bombe run.
	bombeMatch(t, "Multiple Bombe", "BBY", `Crib overruns supplied ciphertext`,
		mb(valid3, "", bombeReflB, "THISISATESTMESSAGE")...)
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `Crib cannot be empty`,
		mb(valid3, "", bombeReflB, "")...)
	bombeMatch(t, "Multiple Bombe", "BBYFLTHHYIJQAYBBYS", `Offset cannot be negative`,
		"User defined", valid3, "", bombeReflB, "THISISATESTMESSAGE", -1, false)
}
