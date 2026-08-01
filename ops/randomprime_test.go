package ops

import (
	"math/big"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/core"
)

// The operation produces a different number every time, so there are no fixture
// values to compare against; what is checked is that every answer keeps the
// promises the operation makes.

// primeFromReport reads the operation's output back into a number.
func primeFromReport(t *testing.T, out string) *big.Int {
	t.Helper()
	n := new(big.Int)
	text, hex := strings.CutPrefix(out, "0x")
	base := 10
	if hex {
		base = 16
	}
	if _, ok := n.SetString(text, base); !ok {
		t.Fatalf("cannot read %q as a number", out)
	}
	return n
}

// TestRandomPrimeIsPrimeOfTheRightSize checks the two promises that matter: the
// answer is prime, and it has exactly the number of bits asked for. Upstream
// only manages the second when the bit length is a multiple of eight — it sets
// a bit to establish the minimum but never clears the ones above it, so asking
// for 17 bits can give 24 (reported upstream).
func TestRandomPrimeIsPrimeOfTheRightSize(t *testing.T) {
	for _, bits := range []float64{2, 3, 8, 10, 17, 32, 64, 128} {
		t.Run(strings.TrimSuffix(big.NewFloat(bits).Text('f', 0), " "), func(t *testing.T) {
			for range 5 {
				out, err := runOp(t, "Pseudo-Random Prime Generator", "", bits, false, "Decimal")
				if err != nil {
					t.Fatalf("Run: %v", err)
				}
				n := primeFromReport(t, out)
				if got := n.BitLen(); got != int(bits) {
					t.Fatalf("asked for %v bits, got %d (%s)", bits, got, n)
				}
				if !n.ProbablyPrime(40) {
					t.Fatalf("%s is not prime", n)
				}
			}
		})
	}
}

// TestRandomPrimeIsNotAlwaysTheSame checks the answer really does vary.
func TestRandomPrimeIsNotAlwaysTheSame(t *testing.T) {
	seen := map[string]bool{}
	for range 8 {
		out, err := runOp(t, "Pseudo-Random Prime Generator", "", 64.0, false, "Decimal")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		seen[out] = true
	}
	if len(seen) == 1 {
		t.Error("every answer was the same number")
	}
}

// TestRandomPrimeHexadecimal checks the other output format.
func TestRandomPrimeHexadecimal(t *testing.T) {
	out, err := runOp(t, "Pseudo-Random Prime Generator", "", 64.0, false, "Hexadecimal")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.HasPrefix(out, "0x") {
		t.Fatalf("got %q, want it to begin 0x", out)
	}
	n := primeFromReport(t, out)
	if n.BitLen() != 64 || !n.ProbablyPrime(40) {
		t.Errorf("%s is not a 64-bit prime", n)
	}
}

// TestRandomPrimeCryptoGrade checks the more thorough setting still answers.
func TestRandomPrimeCryptoGrade(t *testing.T) {
	out, err := runOp(t, "Pseudo-Random Prime Generator", "", 128.0, true, "Decimal")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if n := primeFromReport(t, out); n.BitLen() != 128 || !n.ProbablyPrime(40) {
		t.Errorf("%s is not a 128-bit prime", n)
	}
}

// TestRandomPrimeRefusals covers the two bit lengths the operation will not
// work with.
func TestRandomPrimeRefusals(t *testing.T) {
	// The lower bound is declared on the argument, so it is refused when the
	// arguments are read rather than when the operation runs; a recipe surfaces
	// that as an error, whereas runOp treats it as a broken call.
	tooSmall := core.Recipe{{Op: "Pseudo-Random Prime Generator", Args: []any{1.0, false, "Decimal"}}}
	if _, err := tooSmall.Execute(core.NewDish(nil, core.TypeString)); err == nil {
		t.Error("a bit length below two should be refused")
	}
	_, err := runOp(t, "Pseudo-Random Prime Generator", "", 4097.0, false, "Decimal")
	if err == nil || !strings.Contains(err.Error(), "limited to 4096 bits") {
		t.Errorf("got %v, want the length to be refused as too large", err)
	}
}

// TestRandomPrimeGivesUp checks the bound on the search: if candidates never
// turn out to be prime, the operation says so rather than running for ever.
func TestRandomPrimeGivesUp(t *testing.T) {
	never := func(int) *big.Int { return big.NewInt(4) } // never prime
	if _, err := randomPrime(64, 7, never); err == nil ||
		!strings.Contains(err.Error(), "Failed to generate prime") {
		t.Errorf("got %v, want it to give up", err)
	}
}

// TestRandomOddOfSize checks the candidate generator directly: exactly the
// number of bits asked for, and always odd.
func TestRandomOddOfSize(t *testing.T) {
	for _, bits := range []int{2, 3, 7, 8, 9, 17, 64, 521} {
		for range 20 {
			n := randomOddOfSize(bits)
			if n.BitLen() != bits {
				t.Fatalf("asked for %d bits, got %d (%s)", bits, n.BitLen(), n)
			}
			if n.Bit(0) != 1 {
				t.Fatalf("%s is not odd", n)
			}
		}
	}
}
