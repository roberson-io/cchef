package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Alphabet and translation option values (BACON_ALPHABETS keys / BACON_TRANSLATIONS).
const (
	baconStd  = "Standard (I=J and U=V)"
	baconCmpl = "Complete"
	bacon01   = "0/1"
	baconAB   = "A/B"
	baconCase = "Case"
	baconAMNZ = "A-M/N-Z first letter"
)

// dec builds a Bacon Cipher Decode recipe: [alphabet, translation, invert].
func baconDec(alpha, translation string, invert bool) core.Recipe {
	return core.Recipe{{Op: "Bacon Cipher Decode", Args: []any{alpha, translation, invert}}}
}

// enc builds a Bacon Cipher Encode recipe: [alphabet, translation, keep, invert].
func baconEnc(alpha, translation string, keep, invert bool) core.Recipe {
	return core.Recipe{{Op: "Bacon Cipher Encode", Args: []any{alpha, translation, keep, invert}}}
}

// Cases transcribed from ../CyberChef/tests/operations/tests/BaconCipher.mjs.
func TestBaconDecodeFixtures(t *testing.T) {
	runCases(t, []opCase{
		{"Bacon Decode: no input", "", "", baconDec(baconStd, bacon01, false)},
		{
			"Bacon Decode: reduced alphabet 0/1",
			"00011 00100 00010 01101 00011 01000 01100 00110 00001 00000 00010 01101 01100 10100 01101 10000 01001 10001",
			"DECODINGBACONWORKS", baconDec(baconStd, bacon01, false),
		},
		{
			"Bacon Decode: reduced alphabet 0/1 inverse",
			"11100 11011 11101 10010 11100 10111 10011 11001 11110 11111 11101 10010 10011 01011 10010 01111 10110 01110",
			"DECODINGBACONWORKS", baconDec(baconStd, bacon01, true),
		},
		{
			"Bacon Decode: reduced alphabet A/B lower case",
			"aaabb aabaa aaaba abbab aaabb abaaa abbaa aabba aaaab aaaaa aaaba abbab abbaa babaa abbab baaaa abaab baaab",
			"DECODINGBACONWORKS", baconDec(baconStd, baconAB, false),
		},
		{
			"Bacon Decode: reduced alphabet A/B lower case inverse",
			"bbbaa bbabb bbbab baaba bbbaa babbb baabb bbaab bbbba bbbbb bbbab baaba baabb ababb baaba abbbb babba abbba",
			"DECODINGBACONWORKS", baconDec(baconStd, baconAB, true),
		},
		{
			"Bacon Decode: reduced alphabet A/B upper case",
			"AAABB AABAA AAABA ABBAB AAABB ABAAA ABBAA AABBA AAAAB AAAAA AAABA ABBAB ABBAA BABAA ABBAB BAAAA ABAAB BAAAB",
			"DECODINGBACONWORKS", baconDec(baconStd, baconAB, false),
		},
		{
			"Bacon Decode: reduced alphabet A/B upper case inverse",
			"BBBAA BBABB BBBAB BAABA BBBAA BABBB BAABB BBAAB BBBBA BBBBB BBBAB BAABA BAABB ABABB BAABA ABBBB BABBA ABBBA",
			"DECODINGBACONWORKS", baconDec(baconStd, baconAB, true),
		},
		{
			"Bacon Decode: reduced alphabet case code (Case)",
			"thiS IsaN exampLe oF ThE bacON cIpher WIth upPPercasE letters tRanSLaTiNG to OnEs anD LoWErcase To zERoes. KS",
			"DECODINGBACONWORKS", baconDec(baconStd, baconCase, false),
		},
		{
			"Bacon Decode: reduced alphabet case code (Case) inverse",
			"THIs iS An EXAMPlE Of tHe BACon CiPHER wiTH UPppERCASe LETTERS TrANslAtIng TO oNeS ANd lOweRCASE tO ZerOES. ks",
			"DECODINGBACONWORKS", baconDec(baconStd, baconCase, true),
		},
		{
			"Bacon Decode: reduced alphabet (AMNZ)",
			"A little example of the Bacon Cipher to be decoded. It is a working example and shorter than my others, but it anyways works tremendously. And just that's important, correct?",
			"DECODE", baconDec(baconStd, baconAMNZ, false),
		},
		{
			"Bacon Decode: reduced alphabet (AMNZ) inverse",
			"Well, there's now another example which will be not only strange to read but sound weird for everyone not knowing what the thing is about. Nevertheless, works great out of the box.",
			"DECODE", baconDec(baconStd, baconAMNZ, true),
		},
		{
			"Bacon Decode: complete alphabet 0/1",
			"00011 00100 00010 01110 00011 01000 01101 00110 00001 00000 00010 01110 01101 10110 01110 10001 01010 10010",
			"DECODINGBACONWORKS", baconDec(baconCmpl, bacon01, false),
		},
		{
			"Bacon Decode: complete alphabet 0/1 inverse",
			"11100 11011 11101 10001 11100 10111 10010 11001 11110 11111 11101 10001 10010 01001 10001 01110 10101 01101",
			"DECODINGBACONWORKS", baconDec(baconCmpl, bacon01, true),
		},
		{
			"Bacon Decode: complete alphabet A/B lower case",
			"aaabb aabaa aaaba abbba aaabb abaaa abbab aabba aaaab aaaaa aaaba abbba abbab babba abbba baaab ababa baaba",
			"DECODINGBACONWORKS", baconDec(baconCmpl, baconAB, false),
		},
		{
			"Bacon Decode: complete alphabet A/B lower case inverse",
			"bbbaa bbabb bbbab baaab bbbaa babbb baaba bbaab bbbba bbbbb bbbab baaab baaba abaab baaab abbba babab abbab",
			"DECODINGBACONWORKS", baconDec(baconCmpl, baconAB, true),
		},
		{
			"Bacon Decode: complete alphabet A/B upper case",
			"AAABB AABAA AAABA ABBBA AAABB ABAAA ABBAB AABBA AAAAB AAAAA AAABA ABBBA ABBAB BABBA ABBBA BAAAB ABABA BAABA",
			"DECODINGBACONWORKS", baconDec(baconCmpl, baconAB, false),
		},
		{
			"Bacon Decode: complete alphabet A/B upper case inverse",
			"BBBAA BBABB BBBAB BAAAB BBBAA BABBB BAABA BBAAB BBBBA BBBBB BBBAB BAAAB BAABA ABAAB BAAAB ABBBA BABAB ABBAB",
			"DECODINGBACONWORKS", baconDec(baconCmpl, baconAB, true),
		},
		{
			"Bacon Decode: complete alphabet case code (Case)",
			"thiS IsaN exampLe oF THe bacON cIpher WItH upPPercasE letters tRanSLAtiNG tO OnES anD LOwErcaSe To ZeRoeS. kS",
			"DECODINGBACONWORKS", baconDec(baconCmpl, baconCase, false),
		},
		{
			"Bacon Decode: complete alphabet case code (Case) inverse",
			"THIs iSAn EXAMPlE Of thE BACon CiPHER wiTh UPppERCASe LETTERS TrANslaTIng To zEroES and LoWERcAsE tO oNEs. Ks",
			"DECODINGBACONWORKS", baconDec(baconCmpl, baconCase, true),
		},
		{
			"Bacon Decode: complete alphabet (AMNZ)",
			"A little example of the Bacon Cipher to be decoded. It is a working example and shorter than the first, but it anyways works tremendously. And just that's important, correct?",
			"DECODE", baconDec(baconCmpl, baconAMNZ, false),
		},
		{
			"Bacon Decode: complete alphabet (AMNZ) inverse",
			"Well, there's now another example   which will be not only strange to read but sound weird for everyone knowing nothing what the thing is about. Nevertheless, works great out of the box. ",
			"DECODE", baconDec(baconCmpl, baconAMNZ, true),
		},
		// Out-of-range 5-bit codes map to "?" (oracle-verified): the Standard
		// alphabet has 24 letters so codes 24-31 are "?", and a trailing partial
		// group is dropped.
		{
			"Bacon Decode: out-of-range code yields ?",
			"11000 00000 11111", "?A?", baconDec(baconStd, bacon01, false),
		},
		{
			"Bacon Decode: Complete out-of-range code yields ?",
			"11010 11001", "?Z", baconDec(baconCmpl, bacon01, false),
		},
	})
}

// Cases transcribed from ../CyberChef/tests/operations/tests/BaconCipher.mjs.
func TestBaconEncodeFixtures(t *testing.T) {
	fox := "There's a fox, and it jumps over the fence."
	runCases(t, []opCase{
		{"Bacon Encode: no input", "", "", baconEnc(baconStd, bacon01, false, false)},
		{
			"Bacon Encode: reduced alphabet 0/1", fox,
			"10010 00111 00100 10000 00100 10001 00000 00101 01101 10101 00000 01100 00011 01000 10010 01000 10011 01011 01110 10001 01101 10011 00100 10000 10010 00111 00100 00101 00100 01100 00010 00100",
			baconEnc(baconStd, bacon01, false, false),
		},
		{
			"Bacon Encode: reduced alphabet 0/1 inverse", fox,
			"01101 11000 11011 01111 11011 01110 11111 11010 10010 01010 11111 10011 11100 10111 01101 10111 01100 10100 10001 01110 10010 01100 11011 01111 01101 11000 11011 11010 11011 10011 11101 11011",
			baconEnc(baconStd, bacon01, false, true),
		},
		{
			"Bacon Encode: reduced alphabet 0/1, keeping extra characters", fox,
			"1001000111001001000000100'10001 00000 001010110110101, 000000110000011 0100010010 0100010011010110111010001 01101100110010010000 100100011100100 0010100100011000001000100.",
			baconEnc(baconStd, bacon01, true, false),
		},
		{
			"Bacon Encode: reduced alphabet 0/1 inverse, keeping extra characters", fox,
			"0110111000110110111111011'01110 11111 110101001001010, 111111001111100 1011101101 1011101100101001000101110 10010011001101101111 011011100011011 1101011011100111110111011.",
			baconEnc(baconStd, bacon01, true, true),
		},
		{
			"Bacon Encode: reduced alphabet A/B", fox,
			"BAABA AABBB AABAA BAAAA AABAA BAAAB AAAAA AABAB ABBAB BABAB AAAAA ABBAA AAABB ABAAA BAABA ABAAA BAABB ABABB ABBBA BAAAB ABBAB BAABB AABAA BAAAA BAABA AABBB AABAA AABAB AABAA ABBAA AAABA AABAA",
			baconEnc(baconStd, baconAB, false, false),
		},
		{
			"Bacon Encode: reduced alphabet A/B inverse", fox,
			"ABBAB BBAAA BBABB ABBBB BBABB ABBBA BBBBB BBABA BAABA ABABA BBBBB BAABB BBBAA BABBB ABBAB BABBB ABBAA BABAA BAAAB ABBBA BAABA ABBAA BBABB ABBBB ABBAB BBAAA BBABB BBABA BBABB BAABB BBBAB BBABB",
			baconEnc(baconStd, baconAB, false, true),
		},
		{
			"Bacon Encode: reduced alphabet A/B, keeping extra characters", fox,
			"BAABAAABBBAABAABAAAAAABAA'BAAAB AAAAA AABABABBABBABAB, AAAAAABBAAAAABB ABAAABAABA ABAAABAABBABABBABBBABAAAB ABBABBAABBAABAABAAAA BAABAAABBBAABAA AABABAABAAABBAAAAABAAABAA.",
			baconEnc(baconStd, baconAB, true, false),
		},
		{
			"Bacon Encode: reduced alphabet A/B inverse, keeping extra characters", fox,
			"ABBABBBAAABBABBABBBBBBABB'ABBBA BBBBB BBABABAABAABABA, BBBBBBAABBBBBAA BABBBABBAB BABBBABBAABABAABAAABABBBA BAABAABBAABBABBABBBB ABBABBBAAABBABB BBABABBABBBAABBBBBABBBABB.",
			baconEnc(baconStd, baconAB, true, true),
		},
		{
			"Bacon Encode: complete alphabet 0/1", fox,
			"10011 00111 00100 10001 00100 10010 00000 00101 01110 10111 00000 01101 00011 01000 10011 01001 10100 01100 01111 10010 01110 10101 00100 10001 10011 00111 00100 00101 00100 01101 00010 00100",
			baconEnc(baconCmpl, bacon01, false, false),
		},
		{
			"Bacon Encode: complete alphabet 0/1 inverse", fox,
			"01100 11000 11011 01110 11011 01101 11111 11010 10001 01000 11111 10010 11100 10111 01100 10110 01011 10011 10000 01101 10001 01010 11011 01110 01100 11000 11011 11010 11011 10010 11101 11011",
			baconEnc(baconCmpl, bacon01, false, true),
		},
		{
			"Bacon Encode: complete alphabet 0/1, keeping extra characters", fox,
			"1001100111001001000100100'10010 00000 001010111010111, 000000110100011 0100010011 0100110100011000111110010 01110101010010010001 100110011100100 0010100100011010001000100.",
			baconEnc(baconCmpl, bacon01, true, false),
		},
		{
			"Bacon Encode: complete alphabet 0/1 inverse, keeping extra characters", fox,
			"0110011000110110111011011'01101 11111 110101000101000, 111111001011100 1011101100 1011001011100111000001101 10001010101101101110 011001100011011 1101011011100101110111011.",
			baconEnc(baconCmpl, bacon01, true, true),
		},
		{
			"Bacon Encode: complete alphabet A/B", fox,
			"BAABB AABBB AABAA BAAAB AABAA BAABA AAAAA AABAB ABBBA BABBB AAAAA ABBAB AAABB ABAAA BAABB ABAAB BABAA ABBAA ABBBB BAABA ABBBA BABAB AABAA BAAAB BAABB AABBB AABAA AABAB AABAA ABBAB AAABA AABAA",
			baconEnc(baconCmpl, baconAB, false, false),
		},
		{
			"Bacon Encode: complete alphabet A/B inverse", fox,
			"ABBAA BBAAA BBABB ABBBA BBABB ABBAB BBBBB BBABA BAAAB ABAAA BBBBB BAABA BBBAA BABBB ABBAA BABBA ABABB BAABB BAAAA ABBAB BAAAB ABABA BBABB ABBBA ABBAA BBAAA BBABB BBABA BBABB BAABA BBBAB BBABB",
			baconEnc(baconCmpl, baconAB, false, true),
		},
		{
			"Bacon Encode: complete alphabet A/B, keeping extra characters", fox,
			"BAABBAABBBAABAABAAABAABAA'BAABA AAAAA AABABABBBABABBB, AAAAAABBABAAABB ABAAABAABB ABAABBABAAABBAAABBBBBAABA ABBBABABABAABAABAAAB BAABBAABBBAABAA AABABAABAAABBABAAABAAABAA.",
			baconEnc(baconCmpl, baconAB, true, false),
		},
		{
			"Bacon Encode: complete alphabet A/B inverse, keeping extra characters", fox,
			"ABBAABBAAABBABBABBBABBABB'ABBAB BBBBB BBABABAAABABAAA, BBBBBBAABABBBAA BABBBABBAA BABBAABABBBAABBBAAAAABBAB BAAABABABABBABBABBBA ABBAABBAAABBABB BBABABBABBBAABABBBABBBABB.",
			baconEnc(baconCmpl, baconAB, true, true),
		},
	})
}
