package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// hashRecipe builds a recipe for the operation with the arguments given.
func hashRecipe(length float64, all, total bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract hashes",
		Args: []any{length, all, total},
	}}
}

// TestExtractHashesFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/ExtractHashes.mjs).
func TestExtractHashesFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Extract MD5 hash",
			"The quick brown fox jumps over the lazy dog\n\nMD5: 9e107d9d372bb6826bd81d3542a419d6",
			"9e107d9d372bb6826bd81d3542a419d6",
			hashRecipe(32, false, false),
		},
		{
			"Extract SHA1 hash",
			"The quick brown fox jumps over the lazy dog\n\nSHA1: 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
			"2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
			hashRecipe(40, false, false),
		},
		{
			"Extract SHA256 hash",
			"The quick brown fox jumps over the lazy dog\n\nSHA256: d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			"d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			hashRecipe(64, false, false),
		},
		{
			"Extract SHA512 hash",
			"The quick brown fox jumps over the lazy dog\n\nSHA512: 07e547d9586f6a73f73fbac0435ed76951218fb7d0c8d788a309d785436bbb642e93a252a954f23912547d1e8a3b5ed6e1bfd7097821233fa0538f3db854fee6",
			"07e547d9586f6a73f73fbac0435ed76951218fb7d0c8d788a309d785436bbb642e93a252a954f23912547d1e8a3b5ed6e1bfd7097821233fa0538f3db854fee6",
			hashRecipe(128, false, false),
		},
		{
			"Extract all hashes",
			"The quick brown fox jumps over the lazy dog\n\nMD5: 9e107d9d372bb6826bd81d3542a419d6\nSHA1: 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12\nSHA256: d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			"9e107d9d372bb6826bd81d3542a419d6\n2fd4e1c67a2d28fced849ee1bb76e7391b93eb12\nd7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			hashRecipe(0, true, false),
		},
		{
			"Extract hashes with total count",
			"The quick brown fox jumps over the lazy dog\n\nMD5: 9e107d9d372bb6826bd81d3542a419d6\nSHA1: 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12\nSHA256: d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			"Total Results: 3\n\n9e107d9d372bb6826bd81d3542a419d6\n2fd4e1c67a2d28fced849ee1bb76e7391b93eb12\nd7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592",
			hashRecipe(0, true, true),
		},
	})
}

// TestExtractHashesOptions covers the lengths and switches the fixtures leave
// out, each expectation taken from the CyberChef-server oracle.
func TestExtractHashesOptions(t *testing.T) {
	runCases(t, []opCase{
		{
			"uppercase is not a hash",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"9e107d9d372bb6826bd81d3542a419d6",
			hashRecipe(32, false, false),
		},
		{
			"display total",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"Total Results: 1\n\n9e107d9d372bb6826bd81d3542a419d6",
			hashRecipe(32, false, true),
		},
		{
			"SHA1 length",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
			hashRecipe(40, false, false),
		},
		{
			"four characters",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"1a2b",
			hashRecipe(4, false, false),
		},
		{
			"all lengths, shortest first",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"Total Results: 3\n\n1a2b\n9e107d9d372bb6826bd81d3542a419d6\n2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
			hashRecipe(40, true, true),
		},
		{
			"all lengths on a short input",
			"a b 1a2b deadbeef",
			"Total Results: 4\n\na\nb\n1a2b\ndeadbeef",
			hashRecipe(0, true, true),
		},
		{
			"a length nothing matches",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"Total Results: 0\n\n",
			hashRecipe(64, false, true),
		},
		{
			"a negative length finds nothing",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"Total Results: 0\n\n",
			hashRecipe(-2, false, true),
		},
		{
			"a fractional length finds nothing",
			"MD5 9e107d9d372bb6826bd81d3542a419d6 SHA1 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 UPPER 9E107D9D372BB6826BD81D3542A419D6 crc 1a2b",
			"Total Results: 0\n\n",
			hashRecipe(2.5, false, true),
		},
	})
}
