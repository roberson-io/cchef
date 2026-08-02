package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// bt is a literal backtick (0x60), spliced into raw-string fixtures that would
// otherwise be unrepresentable in a Go raw string literal.
const bt = "`"

// Transcribed from CyberChef's tests/operations/tests/Hexdump.mjs.
func TestHexdumpFixtures(t *testing.T) {
	roundTrip := core.Recipe{
		{Op: "To Hexdump", Args: []any{float64(16), false, false}},
		{Op: "From Hexdump", Args: []any{}},
	}
	fromHexdump := core.Recipe{{Op: "From Hexdump", Args: []any{}}}
	all := allBytes()

	// To Hexdump exact output for ALL_BYTES (offset + hex + Latin-1 ASCII column).
	toHexAllBytes := "00000000  00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f  |................|\n" +
		"00000010  10 11 12 13 14 15 16 17 18 19 1a 1b 1c 1d 1e 1f  |................|\n" +
		`00000020  20 21 22 23 24 25 26 27 28 29 2a 2b 2c 2d 2e 2f  | !"#$%&'()*+,-./|` + "\n" +
		"00000030  30 31 32 33 34 35 36 37 38 39 3a 3b 3c 3d 3e 3f  |0123456789:;<=>?|\n" +
		"00000040  40 41 42 43 44 45 46 47 48 49 4a 4b 4c 4d 4e 4f  |@ABCDEFGHIJKLMNO|\n" +
		`00000050  50 51 52 53 54 55 56 57 58 59 5a 5b 5c 5d 5e 5f  |PQRSTUVWXYZ[\]^_|` + "\n" +
		"00000060  60 61 62 63 64 65 66 67 68 69 6a 6b 6c 6d 6e 6f  |" + bt + "abcdefghijklmno|\n" +
		"00000070  70 71 72 73 74 75 76 77 78 79 7a 7b 7c 7d 7e 7f  |pqrstuvwxyz{|}~.|\n" +
		"00000080  80 81 82 83 84 85 86 87 88 89 8a 8b 8c 8d 8e 8f  |................|\n" +
		"00000090  90 91 92 93 94 95 96 97 98 99 9a 9b 9c 9d 9e 9f  |................|\n" +
		"000000a0  a0 a1 a2 a3 a4 a5 a6 a7 a8 a9 aa ab ac ad ae af  | ¡¢£¤¥¦§¨©ª«¬.®¯|\n" +
		"000000b0  b0 b1 b2 b3 b4 b5 b6 b7 b8 b9 ba bb bc bd be bf  |°±²³´µ¶·¸¹º»¼½¾¿|\n" +
		"000000c0  c0 c1 c2 c3 c4 c5 c6 c7 c8 c9 ca cb cc cd ce cf  |ÀÁÂÃÄÅÆÇÈÉÊËÌÍÎÏ|\n" +
		"000000d0  d0 d1 d2 d3 d4 d5 d6 d7 d8 d9 da db dc dd de df  |ÐÑÒÓÔÕÖ×ØÙÚÛÜÝÞß|\n" +
		"000000e0  e0 e1 e2 e3 e4 e5 e6 e7 e8 e9 ea eb ec ed ee ef  |àáâãäåæçèéêëìíîï|\n" +
		"000000f0  f0 f1 f2 f3 f4 f5 f6 f7 f8 f9 fa fb fc fd fe ff  |ðñòóôõö÷øùúûüýþÿ|"

	toHexUTF8 := "00000000  e1 83 9c e1 83 a3 20 e1 83 9e e1 83 90 e1 83 9c  |á..á.£ á..á..á..|\n" +
		"00000010  e1 83 98 e1 83 99 e1 83 90 e1 83 a1              |á..á..á..á.¡|"

	// xxd, Wireshark, 010, and Linux hexdump formats all decode to ALL_BYTES.
	xxd := "00000000: 0001 0203 0405 0607 0809 0a0b 0c0d 0e0f  ................\n" +
		"00000010: 1011 1213 1415 1617 1819 1a1b 1c1d 1e1f  ................\n" +
		`00000020: 2021 2223 2425 2627 2829 2a2b 2c2d 2e2f   !"#$%&'()*+,-./` + "\n" +
		"00000030: 3031 3233 3435 3637 3839 3a3b 3c3d 3e3f  0123456789:;<=>?\n" +
		"00000040: 4041 4243 4445 4647 4849 4a4b 4c4d 4e4f  @ABCDEFGHIJKLMNO\n" +
		`00000050: 5051 5253 5455 5657 5859 5a5b 5c5d 5e5f  PQRSTUVWXYZ[\]^_` + "\n" +
		"00000060: 6061 6263 6465 6667 6869 6a6b 6c6d 6e6f  " + bt + "abcdefghijklmno\n" +
		"00000070: 7071 7273 7475 7677 7879 7a7b 7c7d 7e7f  pqrstuvwxyz{|}~.\n" +
		"00000080: 8081 8283 8485 8687 8889 8a8b 8c8d 8e8f  ................\n" +
		"00000090: 9091 9293 9495 9697 9899 9a9b 9c9d 9e9f  ................\n" +
		"000000a0: a0a1 a2a3 a4a5 a6a7 a8a9 aaab acad aeaf  ................\n" +
		"000000b0: b0b1 b2b3 b4b5 b6b7 b8b9 babb bcbd bebf  ................\n" +
		"000000c0: c0c1 c2c3 c4c5 c6c7 c8c9 cacb cccd cecf  ................\n" +
		"000000d0: d0d1 d2d3 d4d5 d6d7 d8d9 dadb dcdd dedf  ................\n" +
		"000000e0: e0e1 e2e3 e4e5 e6e7 e8e9 eaeb eced eeef  ................\n" +
		"000000f0: f0f1 f2f3 f4f5 f6f7 f8f9 fafb fcfd feff  ................"

	wireshark := "00000000  00 01 02 03 04 05 06 07  08 09 0a 0b 0c 0d 0e 0f ........ ........\n" +
		"00000010  10 11 12 13 14 15 16 17  18 19 1a 1b 1c 1d 1e 1f ........ ........\n" +
		`00000020  20 21 22 23 24 25 26 27  28 29 2a 2b 2c 2d 2e 2f  !"#$%&' ()*+,-./` + "\n" +
		"00000030  30 31 32 33 34 35 36 37  38 39 3a 3b 3c 3d 3e 3f 01234567 89:;<=>?\n" +
		"00000040  40 41 42 43 44 45 46 47  48 49 4a 4b 4c 4d 4e 4f @ABCDEFG HIJKLMNO\n" +
		`00000050  50 51 52 53 54 55 56 57  58 59 5a 5b 5c 5d 5e 5f PQRSTUVW XYZ[\]^_` + "\n" +
		"00000060  60 61 62 63 64 65 66 67  68 69 6a 6b 6c 6d 6e 6f " + bt + "abcdefg hijklmno\n" +
		"00000070  70 71 72 73 74 75 76 77  78 79 7a 7b 7c 7d 7e 7f pqrstuvw xyz{|}~.\n" +
		"00000080  80 81 82 83 84 85 86 87  88 89 8a 8b 8c 8d 8e 8f ........ ........\n" +
		"00000090  90 91 92 93 94 95 96 97  98 99 9a 9b 9c 9d 9e 9f ........ ........\n" +
		"000000A0  a0 a1 a2 a3 a4 a5 a6 a7  a8 a9 aa ab ac ad ae af ........ ........\n" +
		"000000B0  b0 b1 b2 b3 b4 b5 b6 b7  b8 b9 ba bb bc bd be bf ........ ........\n" +
		"000000C0  c0 c1 c2 c3 c4 c5 c6 c7  c8 c9 ca cb cc cd ce cf ........ ........\n" +
		"000000D0  d0 d1 d2 d3 d4 d5 d6 d7  d8 d9 da db dc dd de df ........ ........\n" +
		"000000E0  e0 e1 e2 e3 e4 e5 e6 e7  e8 e9 ea eb ec ed ee ef ........ ........\n" +
		"000000F0  f0 f1 f2 f3 f4 f5 f6 f7  f8 f9 fa fb fc fd fe ff ........ ........\n"

	wiresharkAlt := "0000   00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f\n" +
		"0010   10 11 12 13 14 15 16 17 18 19 1a 1b 1c 1d 1e 1f\n" +
		"0020   20 21 22 23 24 25 26 27 28 29 2a 2b 2c 2d 2e 2f\n" +
		"0030   30 31 32 33 34 35 36 37 38 39 3a 3b 3c 3d 3e 3f\n" +
		"0040   40 41 42 43 44 45 46 47 48 49 4a 4b 4c 4d 4e 4f\n" +
		"0050   50 51 52 53 54 55 56 57 58 59 5a 5b 5c 5d 5e 5f\n" +
		"0060   60 61 62 63 64 65 66 67 68 69 6a 6b 6c 6d 6e 6f\n" +
		"0070   70 71 72 73 74 75 76 77 78 79 7a 7b 7c 7d 7e 7f\n" +
		"0080   80 81 82 83 84 85 86 87 88 89 8a 8b 8c 8d 8e 8f\n" +
		"0090   90 91 92 93 94 95 96 97 98 99 9a 9b 9c 9d 9e 9f\n" +
		"00a0   a0 a1 a2 a3 a4 a5 a6 a7 a8 a9 aa ab ac ad ae af\n" +
		"00b0   b0 b1 b2 b3 b4 b5 b6 b7 b8 b9 ba bb bc bd be bf\n" +
		"00c0   c0 c1 c2 c3 c4 c5 c6 c7 c8 c9 ca cb cc cd ce cf\n" +
		"00d0   d0 d1 d2 d3 d4 d5 d6 d7 d8 d9 da db dc dd de df\n" +
		"00e0   e0 e1 e2 e3 e4 e5 e6 e7 e8 e9 ea eb ec ed ee ef\n" +
		"00f0   f0 f1 f2 f3 f4 f5 f6 f7 f8 f9 fa fb fc fd fe ff"

	linux := "00000000  00 01 02 03 04 05 06 07  08 09 0a 0b 0c 0d 0e 0f  |................|\n" +
		"00000010  10 11 12 13 14 15 16 17  18 19 1a 1b 1c 1d 1e 1f  |................|\n" +
		`00000020  20 21 22 23 24 25 26 27  28 29 2a 2b 2c 2d 2e 2f  | !"#$%&'()*+,-./|` + "\n" +
		"00000030  30 31 32 33 34 35 36 37  38 39 3a 3b 3c 3d 3e 3f  |0123456789:;<=>?|\n" +
		"00000040  40 41 42 43 44 45 46 47  48 49 4a 4b 4c 4d 4e 4f  |@ABCDEFGHIJKLMNO|\n" +
		`00000050  50 51 52 53 54 55 56 57  58 59 5a 5b 5c 5d 5e 5f  |PQRSTUVWXYZ[\]^_|` + "\n" +
		"00000060  60 61 62 63 64 65 66 67  68 69 6a 6b 6c 6d 6e 6f  |" + bt + "abcdefghijklmno|\n" +
		"00000070  70 71 72 73 74 75 76 77  78 79 7a 7b 7c 7d 7e 7f  |pqrstuvwxyz{|}~.|\n" +
		"00000080  80 81 82 83 84 85 86 87  88 89 8a 8b 8c 8d 8e 8f  |................|\n" +
		"00000090  90 91 92 93 94 95 96 97  98 99 9a 9b 9c 9d 9e 9f  |................|\n" +
		"000000a0  a0 a1 a2 a3 a4 a5 a6 a7  a8 a9 aa ab ac ad ae af  |................|\n" +
		"000000b0  b0 b1 b2 b3 b4 b5 b6 b7  b8 b9 ba bb bc bd be bf  |................|\n" +
		"000000c0  c0 c1 c2 c3 c4 c5 c6 c7  c8 c9 ca cb cc cd ce cf  |................|\n" +
		"000000d0  d0 d1 d2 d3 d4 d5 d6 d7  d8 d9 da db dc dd de df  |................|\n" +
		"000000e0  e0 e1 e2 e3 e4 e5 e6 e7  e8 e9 ea eb ec ed ee ef  |................|\n" +
		"000000f0  f0 f1 f2 f3 f4 f5 f6 f7  f8 f9 fa fb fc fd fe ff  |................|\n" +
		"00000100"

	xxdOdd := "00000000: 6162 6364 65                             abcde"

	runCases(t, []opCase{
		{"Hexdump: nothing", "", "", roundTrip},
		{"Hexdump: Hello, World!", "Hello, World!", "Hello, World!", roundTrip},
		{"Hexdump: UTF-8", "ნუ პანიკას", "ნუ პანიკას", roundTrip},
		{"Hexdump: All bytes", all, all, roundTrip},
		{
			"To Hexdump: UTF-8", "ნუ პანიკას", toHexUTF8,
			core.Recipe{{Op: "To Hexdump", Args: []any{float64(16), false, false}}},
		},
		{
			"To Hexdump: All bytes", all, toHexAllBytes,
			core.Recipe{{Op: "To Hexdump", Args: []any{float64(16), false, false}}},
		},
		{"From Hexdump: xxd", xxd, all, fromHexdump},
		{"From Hexdump: xxd format, odd number of bytes", xxdOdd, "abcde", fromHexdump},
		{"From Hexdump: Wireshark", wireshark, all, fromHexdump},
		{"From Hexdump: Wireshark alt", wiresharkAlt, all, fromHexdump},
		{"From Hexdump: Linux hexdump", linux, all, fromHexdump},
	})
}

// TestHexdumpOptions covers the To Hexdump option flags and the 010-editor "h:"
// offset format, verified against the CyberChef-server oracle. The upstream
// fixtures do not exercise these.
func TestHexdumpOptions(t *testing.T) {
	toHex := func(args ...any) core.Recipe {
		return core.Recipe{{Op: "To Hexdump", Args: args}}
	}
	fromHexBytes := func(args ...any) core.Recipe {
		return core.Recipe{
			{Op: "From Hex", Args: []any{"Auto"}},
			{Op: "To Hexdump", Args: args},
		}
	}
	runCases(t, []opCase{
		{
			"To Hexdump: upper case hex", "Hello",
			"00000000  48 65 6C 6C 6F                                   |Hello|",
			toHex(float64(16), true, false, false),
		},
		{
			"To Hexdump: include final length", "Hello",
			"00000000  48 65 6c 6c 6f                                   |Hello|\n00000005",
			toHex(float64(16), false, true, false),
		},
		{
			"To Hexdump: UNIX format dots non-ASCII", "48e94f",
			"00000000  48 e9 4f                                         |H.O|",
			fromHexBytes(float64(16), false, false, true),
		},
		{
			"To Hexdump: non-UNIX keeps Latin-1", "48e94f",
			"00000000  48 e9 4f                                         |HéO|",
			fromHexBytes(float64(16), false, false, false),
		},
		{
			"To Hexdump: narrow width wraps", "abcdef",
			"00000000  61 62 63  |abc|\n00000003  64 65 66  |def|",
			toHex(float64(3), false, false, false),
		},
		{
			"From Hexdump: 010 editor h: offset", "0000h: 61 62 63 64 65  abcde ", "abcde",
			core.Recipe{{Op: "From Hexdump", Args: []any{}}},
		},
	})
}

// TestHexdumpErrors covers To Hexdump's width validation: the non-integer check
// performed by Run, and the [1, maxHexdumpWidth] bounds declared on the ArgDef
// (enforced by CoerceArgs).
func TestHexdumpErrors(t *testing.T) {
	if _, err := runOp(t, "To Hexdump", "H", 1.5, false, false, false); err == nil {
		t.Fatal("expected an error for a non-integer width")
	}
	for _, width := range []float64{0, 70000} {
		if _, err := core.CoerceArgs(ToHexdump{}.Args(), []any{width, false, false, false}); err == nil {
			t.Fatalf("expected width %v to be rejected by the declared bounds", width)
		}
	}

	// Run must also guard width < 1 directly: a caller bypassing CoerceArgs with
	// width 0 would otherwise spin the encoding loop forever.
	in := core.NewDish([]byte("data"), core.TypeArrayBuffer)
	if _, err := (ToHexdump{}).Run(in, []any{float64(0), false, false, false}); err == nil {
		t.Fatal("expected Run to reject width 0 (would otherwise infinite-loop)")
	}
}
