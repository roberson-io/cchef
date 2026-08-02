package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Cases transcribed from CyberChef's tests/operations/tests/Enigma.mjs. The
// fixtures omit the trailing "Strict output" arg, which CyberChef reads as
// undefined (falsy); cchef fills omitted args with the ingredient default
// (true), so the tests pass Strict output = false explicitly to match.
func TestEnigmaFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Enigma: basic wiring", "G", "P",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: rotor position", "A", "T",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "N", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "F", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "W", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: rotor ring setting", "A", "O",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "B", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: rotor ring setting 2", "A", "F",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "N", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "F", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "W", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: stepping", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "UBDZG OWCXL TKSBT MCDLP BMUQO F",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: reflectivity", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "AAAAA AAAAA AAAAA AAAAA AAAAA A",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: double step anomaly", "AAAAA", "EQIBM",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "D", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "U", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: double step anomaly 2", "AAAA", "BRNC",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "E", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "U", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: double step anomaly 3", "AAAAA AAA", "ZEEQI BMG",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "D", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "S", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: ring setting stepping", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "PBMFE BOUBD ZGOWC XLTKS BTXSH I",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "H", "Z", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: ring setting double step", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "TEVFK UTIIW EDWVI JPMVP GDEZS P",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "Q", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "C", "D", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "H", "F", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
		{
			"Enigma: four rotor", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "GZXGX QUSUW JPWVI GVBTU DQZNZ J",
			core.Recipe{
				{Op: "Enigma", Args: []any{"4-rotor", "LEYJVCNIXWPBQMDRTAKZGFUHOS", "A", "X", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "O", "E", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "P", "F", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "D", "Q", "AE BN CK DQ FU GY HW IJ LO MP RX SZ TV", "", false}},
			},
		},
		{
			"Enigma: four rotor 2", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "HZJLP IKWBZ XNCWF FIHWL EROOZ C",
			core.Recipe{
				{Op: "Enigma", Args: []any{"4-rotor", "FSOKANUERHMBTIYCWLQPZXVGJD", "A", "L", "JPGVOUMFYQBENHZRDKASXLICTW<AN", "A", "J", "VZBRGITYUPSDNHLXAWMJQOFECK<A", "M", "G", "ESOVPZJAYQUIRHXLNFTGKDCMWB<K", "W", "U", "AR BD CO EJ FN GT HK IV LM PW QZ SX UY", "", false}},
			},
		},
		{
			"Enigma: plugboard", "AAAAA AAAAA AAAAA AAAAA AAAAA A", "GHLIM OJIUW DKLWM JGNJK DYJVD K",
			core.Recipe{
				{Op: "Enigma", Args: []any{"4-rotor", "FSOKANUERHMBTIYCWLQPZXVGJD", "A", "I", "NZJHGRCXMYSWBOUFAIVLPEKQDT<AN", "I", "V", "ESOVPZJAYQUIRHXLNFTGKDCMWB<K", "O", "O", "FKQHTLXOCBJSPDZRAMEWNIUYGV<AN", "U", "Z", "AE BN CK DQ FU GY HW IJ LO MP RX SZ TV", "WN MJ LX YB FP QD US IH CE GR", false}},
			},
		},
		{
			"Enigma: decryption", "GHLIM OJIUW DKLWM JGNJK DYJVD K", "AAAAA AAAAA AAAAA AAAAA AAAAA A",
			core.Recipe{
				{Op: "Enigma", Args: []any{"4-rotor", "FSOKANUERHMBTIYCWLQPZXVGJD", "A", "I", "NZJHGRCXMYSWBOUFAIVLPEKQDT<AN", "I", "V", "ESOVPZJAYQUIRHXLNFTGKDCMWB<K", "O", "O", "FKQHTLXOCBJSPDZRAMEWNIUYGV<AN", "U", "Z", "AE BN CK DQ FU GY HW IJ LO MP RX SZ TV", "WN MJ LX YB FP QD US IH CE GR", false}},
			},
		},
		{
			"Enigma: decryption 2", "LANOTCTOUARBBFPMHPHGCZXTDYGAHGUFXGEWKBLKGJWLQXXTGPJJAVTOCKZFSLPPQIHZFXOEBWIIEKFZLCLOAQJULJOYHSSMBBGWHZANVOIIPYRBRTDJQDJJOQKCXWDNBBTYVXLYTAPGVEATXSONPNYNQFUDBBHHVWEPYEYDOHNLXKZDNWRHDUWUJUMWWVIIWZXIVIUQDRHYMNCYEFUAPNHOTKHKGDNPSAKNUAGHJZSMJBMHVTREQEDGXHLZWIFUSKDQVELNMIMITHBHDBWVHDFYHJOQIHORTDJDBWXEMEAYXGYQXOHFDMYUXXNOJAZRSGHPLWMLRECWWUTLRTTVLBHYOORGLGOWUXNXHMHYFAACQEKTHSJW", "KRKRALLEXXFOLGENDESISTSOFORTBEKANNTZUGEBENXXICHHABEFOLGELNBEBEFEHLERHALTENXXJANSTERLEDESBISHERIGXNREICHSMARSCHALLSJGOERINGJSETZTDERFUEHRERSIEYHVRRGRZSSADMIRALYALSSEINENNACHFOLGEREINXSCHRIFTLSCHEVOLLMACHTUNTERWEGSXABSOFORTSOLLENSIESAEMTLICHEMASSNAHMENVERFUEGENYDIESICHAUSDERGEGENWAERTIGENLAGEERGEBENXGEZXREICHSLEITEIKKTULPEKKJBORMANNJXXOBXDXMMMDURNHFKSTXKOMXADMXUUUBOOIEXKP",
			core.Recipe{
				{Op: "Enigma", Args: []any{"4-rotor", "LEYJVCNIXWPBQMDRTAKZGFUHOS", "E", "C", "VZBRGITYUPSDNHLXAWMJQOFECK<A", "P", "D", "JPGVOUMFYQBENHZRDKASXLICTW<AN", "E", "S", "FKQHTLXOCBJSPDZRAMEWNIUYGV<AN", "L", "Z", "AR BD CO EJ FN GT HK IV LM PW QZ SX UY", "AE BF CM DQ HU JN LX PR SZ VW", false}},
			},
		},
		{
			"Enigma: non-alphabet drop", "Hello, world. This is a test.", "ILBDA AMTAZ MORNZ DDIOT U",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", true}},
			},
		},
		{
			"Enigma: non-alphabet passthrough", "Hello, world. This is a test.", "ILBDA, AMTAZ. MORN ZD D IOTU.",
			core.Recipe{
				{Op: "Enigma", Args: []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "", false}},
			},
		},
	})
}

// Validation-error cases from Enigma.mjs (expectedOutput is the error message).
func TestEnigmaValidation(t *testing.T) {
	const input = "Hello, world. This is a test."
	cases := []struct {
		name    string
		args    []any
		wantErr string
	}{
		{"Enigma: rotor validation 1", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQ", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", ""}, "Rotor wiring must be 26 unique uppercase letters"},
		{"Enigma: rotor validation 2", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQo", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", ""}, "Rotor wiring must be 26 unique uppercase letters"},
		{"Enigma: rotor validation 3", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQA", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", ""}, "Rotor wiring must have each letter exactly once"},
		{"Enigma: rotor validation 4", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<RR", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", ""}, "Rotor steps must be unique"},
		{"Enigma: rotor validation 5", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<a", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO TZ VW", ""}, "Rotor steps must be 0-26 unique uppercase letters"},
		{"Enigma: reflector validation 1", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "A", "AY BR CU DH EQ FS GL IP JX KN MO", ""}, "Reflector must have exactly 13 pairs covering every letter"},
		{"Enigma: reflector validation 2", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "A", "AA BR CU DH EQ FS GL IP JX KN MO TZ VV WY", ""}, "Reflector must have exactly 13 pairs covering every letter"},
		{"Enigma: reflector validation 3", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "A", "AY AR CU DH EQ FS GL IP JX KN MO TZ", ""}, "Reflector connects A more than once"},
		{"Enigma: reflector validation 4", []any{"3-rotor", "", "A", "A", "EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A", "AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A", "BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "A", "AYBR CU DH EQ FS GL IP JX KN MO TZ", ""}, "Reflector must be a whitespace-separated list of uppercase letter pairs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := runOp(t, "Enigma", input, c.args...)
			if err == nil {
				t.Fatalf("expected error %q, got nil", c.wantErr)
			}
			if err.Error() != c.wantErr {
				t.Fatalf("got %q\nwant %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestEnigmaRunErrors covers error paths reachable through the operation.
func TestEnigmaRunErrors(t *testing.T) {
	base := func() []any {
		return []any{
			"3-rotor", "", "A", "A",
			"EKMFLGDQVZNTOWYHXUSPAIBRCJ<R", "A", "A",
			"AJDKSIRUXBLHWTMCQGZNPYFVOE<F", "A", "A",
			"BDFHJLCPRTXVZNYEIWGAKMUSQO<W", "A", "Z",
			"AY BR CU DH EQ FS GL IP JX KN MO TZ VW", "",
		}
	}
	cases := []struct {
		name    string
		mut     func([]any)
		wantErr string
	}{
		{"empty rotor", func(a []any) { a[4] = "" }, "Rotor 1 must be provided."},
		{"bad plugboard format", func(a []any) { a[14] = "ABC" }, "Plugboard must be a whitespace-separated list of uppercase letter pairs"},
		{"plugboard connects first twice", func(a []any) { a[14] = "AB AC" }, "Plugboard connects A more than once"},
		{"plugboard connects second twice", func(a []any) { a[14] = "AB CB" }, "Plugboard connects B more than once"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := base()
			c.mut(a)
			_, err := runOp(t, "Enigma", "TEST", a...)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("got %v\nwant %q", err, c.wantErr)
			}
		})
	}
}

// TestEnigmaHelpers covers branches unreachable through the operation (the CLI's
// letter option list guards ring/initial values; the 3/4-rotor logic guarantees
// a valid rotor count).
func TestEnigmaHelpers(t *testing.T) {
	const wiring = "EKMFLGDQVZNTOWYHXUSPAIBRCJ"
	if _, err := newEnigmaRotor(wiring, "R", "AA", "A"); err == nil || err.Error() != "Rotor ring setting must be exactly one uppercase letter" {
		t.Fatalf("ring: %v", err)
	}
	if _, err := newEnigmaRotor(wiring, "R", "A", "1"); err == nil || err.Error() != "Rotor initial position must be exactly one uppercase letter" {
		t.Fatalf("initial: %v", err)
	}
}
