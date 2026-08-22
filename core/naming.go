package core

import "strings"

// kebabFold maps accented Latin letters to their ASCII base so an op name like
// "Vigenère Encode" yields the plain-ASCII subcommand "vigenere-encode".
var kebabFold = map[rune]rune{
	'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a', 'ç': 'c',
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e', 'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i',
	'ñ': 'n', 'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o', 'ø': 'o',
	'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u', 'ý': 'y', 'ÿ': 'y',
}

// Kebab converts an operation name to a CLI subcommand name: lower-cased, with
// accented Latin letters folded to ASCII, spaces and separators collapsed to
// single hyphens, and other punctuation dropped (e.g. "To Base64" -> "to-base64",
// "Find / Replace" -> "find-replace", "Vigenère Encode" -> "vigenere-encode",
// "XPRESS LZ77+Huffman Decompress" -> "xpress-lz77-huffman-decompress").
func Kebab(name string) string {
	var sb strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(name) {
		if folded, ok := kebabFold[r]; ok {
			r = folded
		}
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			sb.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_' || r == '/' || r == '+':
			if !prevHyphen && sb.Len() > 0 {
				sb.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.TrimRight(sb.String(), "-")
}
