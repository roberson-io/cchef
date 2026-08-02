package ops

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(RandomPrime{})
}

// Bit lengths the operation will work between: two is the smallest number that
// can be prime, and the upper end keeps the search quick enough to wait for.
const (
	randomPrimeMinBits = 2
	randomPrimeMaxBits = 4096
)

// How thoroughly a candidate is tested. Upstream calls these seven and forty
// rounds of Miller-Rabin; Go's test does that many and a Lucas test besides, so
// a candidate that passes here has cleared a higher bar than upstream's.
const (
	randomPrimeRounds       = 7
	randomPrimeCryptoRounds = 40
)

// RandomPrime generates a probable prime of a given size.
type RandomPrime struct{}

// Meta returns the operation metadata.
func (RandomPrime) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Pseudo-Random Prime Generator",
		Module:      "Crypto",
		Description: "Generates a random probable prime number of specified bit length using the Miller-Rabin primality test.\n\nPrimality guarantee:\nFor small numbers the result is guaranteed prime. For larger numbers the test is probabilistic:\n- Standard (7 rounds)\n- Crypto grade (40 rounds)\nCrypto grade is recommended for cryptographic applications (RSA, Diffie-Hellman, etc.).",
		InfoURL:     "https://wikipedia.org/wiki/Miller-Rabin_primality_test",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the size wanted, how thoroughly to test, and how to write the
// answer.
func (RandomPrime) Args() []core.ArgDef {
	minBits := float64(randomPrimeMinBits)
	return []core.ArgDef{
		{Name: "Bit length", Type: core.ArgNumber, Integer: true, Value: 512, Min: &minBits},
		{Name: "Crypto grade", Type: core.ArgBoolean, Value: false},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Decimal", "Hexadecimal"}},
	}
}

// Run generates the prime.
func (RandomPrime) Run(in *core.Dish, args []any) (*core.Dish, error) {
	// The lower bound is declared on the argument, so a smaller length is
	// refused before the operation runs; only the upper one is checked here.
	bits := int(args[0].(float64))
	if bits > randomPrimeMaxBits {
		return nil, fmt.Errorf("Bit length limited to %d bits for performance reasons", randomPrimeMaxBits) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}

	rounds := randomPrimeRounds
	if args[1].(bool) {
		rounds = randomPrimeCryptoRounds
	}

	n, err := randomPrime(bits, rounds, randomOddOfSize)
	if err != nil {
		return nil, err
	}
	if args[2].(string) == "Hexadecimal" {
		return core.NewDish([]byte("0x"+n.Text(16)), core.TypeString), nil
	}
	return core.NewDish([]byte(n.String()), core.TypeString), nil
}

// randomPrimeAttempts bounds the search so a hopeless request stops rather than
// running for ever.
const randomPrimeAttempts = 10000

// randomPrime draws odd numbers of exactly the wanted size until one is
// probably prime. The source of candidates is a parameter so the giving-up path
// can be exercised without waiting for an improbable run of luck.
func randomPrime(bits, rounds int, draw func(int) *big.Int) (*big.Int, error) {
	for attempt := 0; attempt <= randomPrimeAttempts; attempt++ {
		n := draw(bits)
		if n.ProbablyPrime(rounds) {
			return n, nil
		}
	}
	return nil, fmt.Errorf("Failed to generate prime after %d attempts. Try a different bit length.", randomPrimeAttempts) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
}

// randomOddOfSize draws a random odd number of exactly the given number of
// bits: the top bit is set so it is not shorter, every bit above it is cleared
// so it is not longer, and the bottom bit is set because no even number above
// two is prime.
//
// Upstream only sets the top bit and leaves the ones above it as they fell, so
// asking it for a length that is not a whole number of bytes can give a longer
// number than asked for — 17 bits can come back 24 bits long.
func randomOddOfSize(bits int) *big.Int {
	if bits == randomPrimeMinBits {
		// Two bits leaves only 2 and 3, and an odd two-bit number is 3.
		return big.NewInt(3)
	}
	buf := make([]byte, (bits+7)/8)
	// crypto/rand.Read never fails; it stops the program rather than returning
	// an error, so there is nothing to check here.
	rand.Read(buf) //nolint:errcheck // documented never to fail

	n := new(big.Int).SetBytes(buf)
	// Keep only the low `bits` bits, then set the top and bottom ones.
	n.And(n, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bits)), big.NewInt(1)))
	n.SetBit(n, bits-1, 1)
	n.SetBit(n, 0, 1)
	return n
}
