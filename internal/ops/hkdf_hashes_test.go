package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestDeriveHKDFKeyAllHashes verifies HKDF output for every supported hash
// function against the CyberChef-server oracle (one vector per hash). It
// exercises each ported hash (SHA0/MD2/HAS160/RIPEMD128-320/Whirlpool*/Snefru).
// hkdfOracle is HKDF(salt=000102...0c, info=f0f1..f9, IKM=0b*22, L=42, with salt) per hash.
var hkdfOracle = map[string]string{
	"MD2":         "2b6de3e67592a93d6ab4aea17b7823afdc2945243a4bc376a5f3c5ac2b5267e68f5c7b32fc70a18d2ab9",
	"MD4":         "f90594f305f28a28890868ebd4033eeae8027ea2005143e76c54e7637afac3cf8827404ab93cd1edf077",
	"MD5":         "b222c9db38d17b2fea8b3bb511c0d6d86049ef481ba7065ca5c6422618ed9cc9144900e2c72b6a863a31",
	"SHA0":        "d6810ac044bdbeeeed291d20f5a167a09384f39f261154dadbab658182c0fc150b06082773d7a9e91d13",
	"SHA1":        "d6000ffb5b50bd3970b260017798fb9c8df9ce2e2c16b6cd709cca07dc3cf9cf26d6c6d750d0aaf5ac94",
	"SHA224":      "2f21cd7cbc818ca5c561b933728e2e08e154a87e1432399a820dee13aa222d0cee6152fa539ab70f8e80",
	"SHA256":      "3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865",
	"SHA384":      "9b5097a86038b805309076a44b3a9f38063e25b516dcbf369f394cfab43685f748b6457763e4f0204fc5",
	"SHA512":      "832390086cda71fb47625bb5ceb168e4c8e26a1a16ed34d9fc7fe92c1481579338da362cb8d9f925d7cb",
	"SHA512/224":  "f8d956e152b0fba831bac400f1a5af54982b91db3d96ae21a75655eff1725f928e491c63f3aedb408296",
	"SHA512/256":  "789a93e567a1861de449342b2d674c0df737fd8adce2a8e1843237c1938ac413044b496ce267a198ebe3",
	"RIPEMD128":   "59fc9e7c82667a818d92c7ba8206e27989aab22c13aad1779bc0e3b9b00481e4e4730b0e6b560f60fb63",
	"RIPEMD160":   "8e2a6e5c36796c02636a4246873f35edf59684f394da0ec847b3643aa1f0059ce97de9843cf9db968a88",
	"RIPEMD256":   "84fe36c95997f94c8079d3ebe1fd7d5bd1dd7817e2b9e0919bcf73c8fbdeb3b49b0f9a610075e5787c1c",
	"RIPEMD320":   "a163974e2bc5655e81525873b1631a820644cefff0d5222ed5dcf844ced0fac2d63741ee968d90eb66e3",
	"HAS160":      "c3c5661661acd627dd98bd0d50bb8e5cac76ddbf0422f34e0621e4a1f0022b2048a05089dc86bed6d765",
	"Whirlpool":   "0d29f74ccd8640f44b0dd9638111c1b5766efed752af358109e2e7c9cd4a28ef2f90b2ad461fba0744d4",
	"Whirlpool-0": "b43a3622ddd2a15b5784cad15c5b5393772870f483b964da9296a426fc72ba7138cca4466ee1611bd5bc",
	"Whirlpool-T": "4c6f2307b6ce3aa5c57bb57d962915d764c151f2b1f652763a68808eaa4572c8f46991944a8313b65adc",
	"Snefru":      "e3c848341731343fcfa4847d20e232b686b803dd09badca120ed38105c1fe47145473379658aff1623d5",
}

func TestDeriveHKDFKeyAllHashes(t *testing.T) {
	for _, h := range hkdfHashOptions {
		out, err := runOp(t, "Derive HKDF key",
			string([]byte{0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b}),
			core.ToggleString{Value: "000102030405060708090a0b0c", Option: "Hex"},
			core.ToggleString{Value: "f0f1f2f3f4f5f6f7f8f9", Option: "Hex"},
			h, "with salt", float64(42))
		if err != nil {
			t.Errorf("%-12s ERROR %v", h, err)
			continue
		}
		if out != hkdfOracle[h] {
			t.Errorf("%-12s MISMATCH\n  got  %s\n  want %s", h, out, hkdfOracle[h])
		}
	}
}

// hkdfHex builds a From Hex → HKDF recipe over an IKM given as hex.
func hkdfHex(ikmHex, salt, info, hasher, mode string, l int) (string, error) {
	r := core.Recipe{
		{Op: "From Hex", Args: []any{"None"}},
		{Op: "Derive HKDF key", Args: []any{
			core.ToggleString{Value: salt, Option: "Hex"},
			core.ToggleString{Value: info, Option: "Hex"},
			hasher, mode, l,
		}},
	}
	out, err := r.Execute(core.NewDish([]byte(ikmHex), core.TypeString))
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// TestDeriveHKDFKeyPaddingBoundaries exercises the block-padding branches: a
// message whose length is 56 mod 64 (md64.pad extra-block path) and a Whirlpool
// message whose length is 32-63 mod 64 (its own extra-block path). Values from
// the oracle.
func TestDeriveHKDFKeyPaddingBoundaries(t *testing.T) {
	cases := []struct {
		name, ikm, hasher, want string
	}{
		{
			"SHA0 len56", strings.Repeat("ab", 56), "SHA0",
			"bce13bca5bd1d07eac7220d12e7cd77aa9446903a3619e31a23e6b671cc31fb91f08dca7fd0f8a915092ee0061fdd1ba",
		},
		{
			"Whirlpool len40", strings.Repeat("ab", 40), "Whirlpool",
			"622807ed49e889675c2e377aab4469b0dd6d5873b35ee3299ecb02eade92755e0e6f72729b6c3b4206c5593d0774c5c9",
		},
		{
			"RIPEMD320 len60", strings.Repeat("ab", 60), "RIPEMD320",
			"a5fb9fa1f6934c684ea1e73dc555ec9078bb0e7a7d63e1a826a1c7e7cc8681791dc08d32eb1a32158f04802d105b8f62",
		},
	}
	for _, c := range cases {
		got, err := hkdfHex(c.ikm, "a1b2c3", "0102030405", c.hasher, "with salt", 48)
		if err != nil || got != c.want {
			t.Errorf("%s: got %q (%v)", c.name, got, err)
		}
	}
}

// TestDeriveHKDFKeyMoreErrors covers the negative-L, invalid-Base64 and
// (defensive) unsupported-hash branches.
func TestDeriveHKDFKeyMoreErrors(t *testing.T) {
	// Negative L.
	if _, err := runOp(t, "Derive HKDF key", "x",
		core.ToggleString{Value: "", Option: "Hex"}, core.ToggleString{Value: "", Option: "Hex"},
		"SHA256", "with salt", float64(-1)); err == nil || !strings.Contains(err.Error(), "L must be non-negative") {
		t.Fatalf("negative L: %v", err)
	}
	// Invalid Base64 salt (cchef's strict decoder).
	if _, err := runOp(t, "Derive HKDF key", "x",
		core.ToggleString{Value: "!!!", Option: "Base64"}, core.ToggleString{Value: "", Option: "Hex"},
		"SHA256", "with salt", float64(16)); err == nil {
		t.Fatal("expected error for invalid Base64 salt")
	}
	// Invalid Base64 info.
	if _, err := runOp(t, "Derive HKDF key", "x",
		core.ToggleString{Value: "", Option: "Hex"}, core.ToggleString{Value: "!!!", Option: "Base64"},
		"SHA256", "with salt", float64(16)); err == nil {
		t.Fatal("expected error for invalid Base64 info")
	}
	// Unsupported hash (defensive branch; reached by calling Run directly with an
	// unvalidated hash name).
	_, err := DeriveHKDFKey{}.Run(core.NewDish([]byte("x"), core.TypeArrayBuffer), []any{
		core.ToggleString{Value: "", Option: "Hex"},
		core.ToggleString{Value: "", Option: "Hex"},
		"BOGUS", "with salt", float64(16),
	})
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unsupported hash: %v", err)
	}
}
