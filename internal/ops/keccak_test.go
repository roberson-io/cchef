package ops

import (
	"crypto/sha3"
	"encoding/hex"
	"hash"
	"testing"

	xsha3 "golang.org/x/crypto/sha3"

	"github.com/roberson-io/cchef/internal/core"
)

// Keccak digests transcribed from CyberChef tests/operations/tests/Hash.mjs
// (input "Hello, World!"). Keccak differs from SHA-3 only in the domain byte.
func TestKeccakFixtures(t *testing.T) {
	const in = "Hello, World!"
	runCases(t, []opCase{
		{
			"Keccak 224", in, "4eaaf0e7a1e400efba71130722e1cb4d59b32afb400e654afec4f8ce",
			core.Recipe{{Op: "Keccak", Args: []any{"224"}}},
		},
		{
			"Keccak 256", in, "acaf3289d7b601cbd114fb36c4d29c85bbfd5e133f14cb355c3fd8d99367964f",
			core.Recipe{{Op: "Keccak", Args: []any{"256"}}},
		},
		{
			"Keccak 384", in, "4d60892fde7f967bcabdc47c73122ae6311fa1f9be90d721da32030f7467a2e3db3f9ccb3c746483f9d2b876e39def17",
			core.Recipe{{Op: "Keccak", Args: []any{"384"}}},
		},
		{
			"Keccak 512", in, "eda765576c84c600ed7f5d97510e92703b61f5215def2a161037fd9dd1f5b6ed4f86ce46073c0e3f34b52de0289e9c618798fff9dd4b1bfe035bdb8645fc6e37",
			core.Recipe{{Op: "Keccak", Args: []any{"512"}}},
		},
		// Default size is 512.
		{
			"Keccak default", in, "eda765576c84c600ed7f5d97510e92703b61f5215def2a161037fd9dd1f5b6ed4f86ce46073c0e3f34b52de0289e9c618798fff9dd4b1bfe035bdb8645fc6e37",
			core.Recipe{{Op: "Keccak"}},
		},
	})
}

// TestKeccakSpongeMatchesXCrypto cross-validates the local Keccak-f sponge in
// Keccak mode (domain 0x01) against the vetted x/crypto legacy-Keccak hashers
// for the sizes they share (256/512). This indirectly validates the 224/384
// path, which shares the same permutation and padding.
func TestKeccakSpongeMatchesXCrypto(t *testing.T) {
	hashers := map[int]func() hash.Hash{
		256: xsha3.NewLegacyKeccak256,
		512: xsha3.NewLegacyKeccak512,
	}
	for size, newH := range hashers {
		for n := 0; n < 400; n += 7 {
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i*53 + 1)
			}
			got := hex.EncodeToString(keccakSum(data, size, domainKeccak))
			h := newH()
			h.Write(data)
			want := hex.EncodeToString(h.Sum(nil))
			if got != want {
				t.Fatalf("Keccak-%d len %d: sponge=%s xcrypto=%s", size, n, got, want)
			}
		}
	}
}

// TestKeccakSpongeMatchesStdlibSHA3 validates the from-scratch Keccak-f sponge:
// in SHA-3 mode (domain byte 0x06) it must reproduce Go's stdlib crypto/sha3
// for every size and a range of input lengths. This pins the permutation
// independently of the Keccak vectors above.
func TestKeccakSpongeMatchesStdlibSHA3(t *testing.T) {
	stdlib := map[int]func() hash.Hash{
		224: func() hash.Hash { return sha3.New224() },
		256: func() hash.Hash { return sha3.New256() },
		384: func() hash.Hash { return sha3.New384() },
		512: func() hash.Hash { return sha3.New512() },
	}
	for size, newH := range stdlib {
		for n := 0; n < 400; n += 7 {
			data := make([]byte, n)
			for i := range data {
				data[i] = byte(i * 31)
			}
			got := hex.EncodeToString(keccakSum(data, size, domainSHA3))
			h := newH()
			h.Write(data)
			want := hex.EncodeToString(h.Sum(nil))
			if got != want {
				t.Fatalf("SHA3-%d len %d: sponge=%s stdlib=%s", size, n, got, want)
			}
		}
	}
}
