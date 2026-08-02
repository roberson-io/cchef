package ops

import (
	"crypto/md5" // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
)

func init() {
	core.Register(JA3Fingerprint{})
}

// JA3Fingerprint generates a JA3 fingerprint from a TLS Client Hello.
type JA3Fingerprint struct{}

// Meta returns the operation metadata.
func (JA3Fingerprint) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JA3 Fingerprint",
		Module:      "Crypto",
		Description: "Generates a JA3 fingerprint to help identify TLS clients based on hashing together values from the Client Hello. Input: a hex stream of the TLS Client Hello packet application layer.",
		InfoURL:     "https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JA3Fingerprint) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Hash digest", "JA3 string", "Full details"}},
	}
}

// Run generates the JA3 fingerprint.
func (JA3Fingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)
	outputFormat := args[1].(string)

	data := fingerprintBytes(in.String(), inputFormat)
	s := bytestream.New(data)

	if s.ReadInt(1) != 0x16 {
		return nil, fingerprintError("not handshake data")
	}
	s.MoveForwardsBy(2) // version
	length := s.ReadInt(2)
	if s.Length() != length+5 {
		return nil, fingerprintError("incorrect handshake length")
	}
	if s.ReadInt(1) != 1 {
		return nil, fingerprintError("not a Client Hello")
	}
	handshakeLength := s.ReadInt(3)
	if s.Length() != handshakeLength+9 {
		return nil, fingerprintError("not enough data in Client Hello")
	}
	helloVersion := s.ReadInt(2)
	s.MoveForwardsBy(32) // random
	sessionIDLength := s.ReadInt(1)
	s.MoveForwardsBy(sessionIDLength)

	cipherSuitesLength := s.ReadInt(2)
	cipherSegment := parseJA3Segment(bytestream.New(s.GetBytes(cipherSuitesLength)), 2)

	compressionMethodsLength := s.ReadInt(1)
	s.MoveForwardsBy(compressionMethodsLength)

	extensionsLength := s.ReadInt(2)
	es := bytestream.New(s.GetBytes(extensionsLength))
	ellipticCurves, ellipticCurvePointFormats := "", ""
	var exts []string
	for es.HasMore() {
		typ := es.ReadInt(2)
		extLength := es.ReadInt(2)
		switch typ {
		case 0x0a: // Elliptic curves
			ecsLen := es.ReadInt(2)
			ellipticCurves = parseJA3Segment(bytestream.New(es.GetBytes(ecsLen)), 2)
		case 0x0b: // Elliptic curve point formats
			ecsLen := es.ReadInt(1)
			ellipticCurvePointFormats = parseJA3Segment(bytestream.New(es.GetBytes(ecsLen)), 1)
		default:
			es.MoveForwardsBy(extLength)
		}
		if !greaseCipherSuites[typ] {
			exts = append(exts, strconv.Itoa(typ))
		}
	}

	ja3Str := strings.Join([]string{
		strconv.Itoa(helloVersion),
		cipherSegment,
		strings.Join(exts, "-"),
		ellipticCurves,
		ellipticCurvePointFormats,
	}, ",")
	ja3Hash := fmt.Sprintf("%x", md5.Sum([]byte(ja3Str))) // #nosec G401 -- MD5/SHA1 is an intentional CyberChef operation, not a security control

	switch outputFormat {
	case "JA3 string":
		return core.NewDish([]byte(ja3Str), core.TypeString), nil
	case "Full details":
		out := fmt.Sprintf("Hash digest:\n%s\n\nFull JA3 string:\n%s\n\nTLS Version:\n%d\nCipher Suites:\n%s\nExtensions:\n%s\nElliptic Curves:\n%s\nElliptic Curve Point Formats:\n%s",
			ja3Hash, ja3Str, helloVersion, cipherSegment, strings.Join(exts, "-"), ellipticCurves, ellipticCurvePointFormats)
		return core.NewDish([]byte(out), core.TypeString), nil
	default: // Hash digest
		return core.NewDish([]byte(ja3Hash), core.TypeString), nil
	}
}
