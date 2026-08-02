package ops

// Shared helpers for the CTPH and SSDEEP fuzzy-hashing operations.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// fuzzyB64 is the base64 alphabet the fuzzy hashers index into.
const fuzzyB64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// fuzzyLevenshtein is the Levenshtein edit distance between two strings.
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
func fuzzySimilarity(d1, d2 string) (float64, error) {
	b1 := fuzzyBlockSize(d1)
	b2 := fuzzyBlockSize(d2)
	if b1 > b2 {
		return fuzzySimilarity(d2, d1)
	}
	switch {
	case abs(b1-b2) > 1:
		// Too far apart in block size to have anything in common, which needs
		// nothing else from either hash.
		return 0, nil
	case b1 == b2:
		return fuzzyScoreParts(d1, 1, d2, 1)
	default:
		return fuzzyScoreParts(d1, 2, d2, 1)
	}
}

// fuzzyBlockSize reads the character a hash starts with as its block size. An
// empty hash counts as zero, which is where JavaScript's charAt of nothing
// lands, so a pair that includes one still scores rather than failing here.
func fuzzyBlockSize(hash string) int {
	if hash == "" {
		return 0
	}
	return strings.IndexByte(fuzzyB64, hash[0])
}

// fuzzyScoreParts scores the named field of each hash, failing when a hash
// does not have that field.
func fuzzyScoreParts(d1 string, n1 int, d2 string, n2 int) (float64, error) {
	p1, err := fuzzyPart(d1, n1)
	if err != nil {
		return 0, err
	}
	p2, err := fuzzyPart(d2, n2)
	if err != nil {
		return 0, err
	}
	return fuzzyMatchScore(p1, p2), nil
}

// fuzzyPart returns the n-th colon-separated field of a hash (0-based).
func fuzzyPart(hash string, n int) (string, error) {
	parts := strings.Split(hash, ":")
	if n >= len(parts) {
		return "", fmt.Errorf("%q is not an SSDEEP hash: it has %d colon-separated parts, and %d are needed",
			hash, len(parts), n+1)
	}
	return parts[n], nil
}

// fuzzyCompare splits the input into exactly two hashes on the delimiter and
// returns their similarity formatted as CyberChef's Number output.
func fuzzyCompare(input, delimName string) (string, error) {
	samples := strings.Split(input, charRep(delimName))
	if len(samples) != 2 {
		return "", errors.New("Incorrect number of samples.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	score, err := fuzzySimilarity(samples[0], samples[1])
	if err != nil {
		return "", err
	}
	return strconv.FormatFloat(score, 'g', -1, 64), nil
}
