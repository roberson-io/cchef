package ops

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
)

// Number-theory helpers shared by the operations that work on whole numbers of
// any size.

// bigIntDecimal and bigIntHex are the two shapes such a number may be written
// in: an optionally signed decimal, or an unsigned hexadecimal with an 0x
// prefix.
var (
	bigIntDecimal = regexp.MustCompile(`^[+-]?[0-9]+$`)
	bigIntHex     = regexp.MustCompile(`^0[xX][0-9a-fA-F]+$`)
)

// parseBigInt reads a whole number written either way, naming the argument it
// came from when it will not read.
func parseBigInt(value, param string) (*big.Int, error) {
	v := strings.TrimSpace(value)
	n := new(big.Int)
	switch {
	case bigIntHex.MatchString(v):
		n.SetString(v[2:], 16)
	case bigIntDecimal.MatchString(v):
		n.SetString(v, 10)
	default:
		return nil, fmt.Errorf("%s must be decimal or hex (0x...)", param) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return n, nil
}

// egcd runs the extended Euclidean algorithm, returning g, x and y such that
// a*x + b*y = g. The division truncates towards zero, as upstream's does, so
// the coefficients come out with the same signs.
func egcd(a, b *big.Int) (g, x, y *big.Int) {
	oldR, r := new(big.Int).Set(a), new(big.Int).Set(b)
	oldS, s := big.NewInt(1), big.NewInt(0)
	oldT, t := big.NewInt(0), big.NewInt(1)

	for r.Sign() != 0 {
		q := new(big.Int).Quo(oldR, r)
		oldR, r = r, new(big.Int).Sub(oldR, new(big.Int).Mul(q, r))
		oldS, s = s, new(big.Int).Sub(oldS, new(big.Int).Mul(q, s))
		oldT, t = t, new(big.Int).Sub(oldT, new(big.Int).Mul(q, t))
	}
	return oldR, oldS, oldT
}

// twoValues works out which two numbers an operation should act on, given two
// arguments either of which may be left blank to take that value from the
// input instead.
func twoValues(input, first, second, firstName, secondName string) (string, string, error) {
	a, b := strings.TrimSpace(first), strings.TrimSpace(second)
	in := strings.TrimSpace(input)

	switch {
	case a != "" && b != "":
		return a, b, nil
	case a == "" && b != "":
		if in == "" {
			return "", "", fmt.Errorf("%s must be defined", firstName) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		return in, b, nil
	case a != "" && b == "":
		if in == "" {
			return "", "", fmt.Errorf("%s must be defined", secondName) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		}
		return a, in, nil
	}
	return "", "", fmt.Errorf("%s and %s must be defined", firstName, secondName) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
}
