package ops

// Shared helpers for the CTPH and SSDEEP fuzzy-hashing operations, ported from
// the ctph.js / ssdeep.js npm packages (their similarity and Levenshtein logic
// is byte-identical, so it lives here).

import (
	"errors"
	"strconv"
	"strings"
)

// fuzzyB64 is the base64 alphabet the fuzzy hashers index into.
const fuzzyB64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// fuzzyLevenshtein is the Levenshtein edit distance between two strings, ported
// from the fast-levenshtein implementation the packages use.
func fuzzyLevenshtein(str1, str2 string) int {
	if str1 == str2 {
		return 0
	}
	if len(str1) == 0 {
		return len(str2)
	}
	if len(str2) == 0 {
		return len(str1)
	}

	prevRow := make([]int, len(str2)+1)
	for i := range prevRow {
		prevRow[i] = i
	}

	nextCol := 0
	for i := range len(str1) {
		nextCol = i + 1
		for j := range len(str2) {
			curCol := nextCol
			sub := 1
			if str1[i] == str2[j] {
				sub = 0
			}
			nextCol = prevRow[j] + sub
			if tmp := curCol + 1; nextCol > tmp {
				nextCol = tmp
			}
			if tmp := prevRow[j+1] + 1; nextCol > tmp {
				nextCol = tmp
			}
			prevRow[j] = curCol
		}
		prevRow[len(str2)] = nextCol
	}
	return nextCol
}

// fuzzyMatchScore scores two signatures as (1 - dist/maxLen) * 100.
func fuzzyMatchScore(s1, s2 string) float64 {
	e := fuzzyLevenshtein(s1, s2)
	maxLen := max(len(s2), len(s1))
	return (1 - float64(e)/float64(maxLen)) * 100
}

// fuzzySimilarity compares two fuzzy hashes, returning a 0–100 score. Identical
// logic in both packages.
func fuzzySimilarity(d1, d2 string) float64 {
	b1 := strings.IndexByte(fuzzyB64, d1[0])
	b2 := strings.IndexByte(fuzzyB64, d2[0])
	if b1 > b2 {
		return fuzzySimilarity(d2, d1)
	}
	switch {
	case abs(b1-b2) > 1:
		return 0
	case b1 == b2:
		return fuzzyMatchScore(fuzzyPart(d1, 1), fuzzyPart(d2, 1))
	default:
		return fuzzyMatchScore(fuzzyPart(d1, 2), fuzzyPart(d2, 1))
	}
}

// fuzzyPart returns the n-th colon-separated field of a hash (0-based).
func fuzzyPart(hash string, n int) string {
	parts := strings.Split(hash, ":")
	return parts[n]
}

// fuzzyCompare splits the input into exactly two hashes on the delimiter and
// returns their similarity formatted as CyberChef's Number output.
func fuzzyCompare(input, delimName string) (string, error) {
	samples := strings.Split(input, charRep(delimName))
	if len(samples) != 2 {
		return "", errors.New("Incorrect number of samples.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	score := fuzzySimilarity(samples[0], samples[1])
	return strconv.FormatFloat(score, 'g', -1, 64), nil
}
