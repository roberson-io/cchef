package ops

import (
	"errors"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
)

func init() {
	core.Register(ObjectIdentifierToHex{})
	core.Register(HexToObjectIdentifier{})
}

var oidStrRe = regexp.MustCompile(`^[0-9.]+$`)

// ObjectIdentifierToHex converts a dotted-decimal OID into the hex of its ASN.1
// content octets. Ported from CyberChef's ObjectIdentifierToHex, which wraps
// jsrsasign's KJUR.asn1.ASN1Util.oidIntToHex.
type ObjectIdentifierToHex struct{}

// Meta returns the operation metadata.
func (ObjectIdentifierToHex) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Object Identifier to Hex",
		Module:      "PublicKey",
		Description: "Converts an object identifier (OID) into a hexadecimal string.",
		InfoURL:     "https://wikipedia.org/wiki/Object_identifier",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ObjectIdentifierToHex) Args() []core.ArgDef { return nil }

// Run encodes the OID string to hex.
func (ObjectIdentifierToHex) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := asn1OidIntToHex(in.String())
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// HexToObjectIdentifier converts the hex of ASN.1 OID content octets back into
// dotted-decimal notation. Ported from CyberChef's HexToObjectIdentifier, which
// wraps jsrsasign's KJUR.asn1.ASN1Util.oidHexToInt after stripping whitespace.
type HexToObjectIdentifier struct{}

// Meta returns the operation metadata.
func (HexToObjectIdentifier) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Hex to Object Identifier",
		Module:      "PublicKey",
		Description: "Converts a hexadecimal string into an object identifier (OID).",
		InfoURL:     "https://wikipedia.org/wiki/Object_identifier",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HexToObjectIdentifier) Args() []core.ArgDef { return nil }

// Run decodes the hex content octets to an OID string.
func (HexToObjectIdentifier) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out := asn1OidHexToInt(stripWhitespace(in.String()))
	return core.NewDish([]byte(out), core.TypeString), nil
}

// asn1OidIntToHex converts a dotted-decimal OID to the hex of its ASN.1 content
// octets.
func asn1OidIntToHex(f string) (string, error) {
	if !oidStrRe.MatchString(f) {
		return "", errors.New("malformed oid string: " + f)
	}
	b := strings.Split(f, ".")
	a0, ok0 := jsnum.ParseInt(b[0], 10)
	a1, ok1 := 0, false
	if len(b) > 1 {
		a1, ok1 = jsnum.ParseInt(b[1], 10)
	}
	// j = parseInt(b[0])*40 + parseInt(b[1]); a missing/empty arc makes j NaN,
	// whose toString(16) is the literal "NaN" (the local pad only touches
	// single-character output).
	var g string
	if !ok0 || !ok1 {
		g = "NaN"
	} else {
		g = oidByteHex(a0*40 + a1)
	}
	rest := []string{}
	if len(b) > 2 {
		rest = b[2:]
	}
	for _, arc := range rest {
		g += asn1OidSubIDToHex(arc)
	}
	return g, nil
}

// asn1OidSubIDToHex encodes one OID arc as base-128 (7 bits per byte, high bit
// set on all but the final byte). Mirrors the inner helper of oidIntToHex.
func asn1OidSubIDToHex(dec string) string {
	k := new(big.Int)
	k.SetString(dec, 10)
	a := k.Text(2)
	l := 7 - len(a)%7
	if l == 7 {
		l = 0
	}
	a = strings.Repeat("0", l) + a
	var n strings.Builder
	for m := 0; m < len(a)-1; m += 7 {
		p := a[m : m+7]
		if m != len(a)-7 {
			p = "1" + p
		}
		v, _ := strconv.ParseInt(p, 2, 0)
		n.WriteString(oidByteHex(int(v)))
	}
	return n.String()
}

// oidByteHex renders a value as hex, padding to two characters only when the
// result is a single character. Mirrors jsrsasign's local integer-to-byte-hex
// helper (which, unlike the general one, pads on length==1 rather than parity).
func oidByteHex(v int) string {
	k := strconv.FormatInt(int64(v), 16)
	if len(k) == 1 {
		k = "0" + k
	}
	return k
}
