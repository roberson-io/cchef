package ops

import (
	"crypto/md5" // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
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

// Run generates the JA3 fingerprint. Ported from CyberChef JA3Fingerprint.mjs.
func (JA3Fingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)
	outputFormat := args[1].(string)

	data := fingerprintBytes(in.String(), inputFormat)
	s := newByteStream(data)

	if s.readInt(1) != 0x16 {
		return nil, fingerprintError("not handshake data")
	}
	s.moveForwardsBy(2) // version
	length := s.readInt(2)
	if s.length() != length+5 {
		return nil, fingerprintError("incorrect handshake length")
	}
	if s.readInt(1) != 1 {
		return nil, fingerprintError("not a Client Hello")
	}
	handshakeLength := s.readInt(3)
	if s.length() != handshakeLength+9 {
		return nil, fingerprintError("not enough data in Client Hello")
	}
	helloVersion := s.readInt(2)
	s.moveForwardsBy(32) // random
	sessionIDLength := s.readInt(1)
	s.moveForwardsBy(sessionIDLength)

	cipherSuitesLength := s.readInt(2)
	cipherSegment := parseJA3Segment(newByteStream(s.getBytes(cipherSuitesLength)), 2)

	compressionMethodsLength := s.readInt(1)
	s.moveForwardsBy(compressionMethodsLength)

	extensionsLength := s.readInt(2)
	es := newByteStream(s.getBytes(extensionsLength))
	ellipticCurves, ellipticCurvePointFormats := "", ""
	var exts []string
	for es.hasMore() {
		typ := es.readInt(2)
		extLength := es.readInt(2)
		switch typ {
		case 0x0a: // Elliptic curves
			ecsLen := es.readInt(2)
			ellipticCurves = parseJA3Segment(newByteStream(es.getBytes(ecsLen)), 2)
		case 0x0b: // Elliptic curve point formats
			ecsLen := es.readInt(1)
			ellipticCurvePointFormats = parseJA3Segment(newByteStream(es.getBytes(ecsLen)), 1)
		default:
			es.moveForwardsBy(extLength)
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
