package ops

import (
	"crypto/md5" // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JA3SFingerprint{})
}

// JA3SFingerprint generates a JA3S fingerprint from a TLS Server Hello.
type JA3SFingerprint struct{}

// Meta returns the operation metadata.
func (JA3SFingerprint) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JA3S Fingerprint",
		Module:      "Crypto",
		Description: "Generates a JA3S fingerprint to help identify TLS servers based on hashing together values from the Server Hello. Input: a hex stream of the TLS Server Hello record application layer.",
		InfoURL:     "https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JA3SFingerprint) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Hash digest", "JA3S string", "Full details"}},
	}
}

// Run generates the JA3S fingerprint. Ported from CyberChef JA3SFingerprint.mjs.
func (JA3SFingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)
	outputFormat := args[1].(string)

	data := fingerprintBytes(in.String(), inputFormat)
	s := newByteStream(data)

	if s.readInt(1) != 0x16 {
		return nil, fingerprintError("not handshake data")
	}
	s.moveForwardsBy(2)
	length := s.readInt(2)
	if s.length() != length+5 {
		return nil, fingerprintError("incorrect handshake length")
	}
	if s.readInt(1) != 2 {
		return nil, fingerprintError("not a Server Hello")
	}
	handshakeLength := s.readInt(3)
	if s.length() != handshakeLength+9 {
		return nil, fingerprintError("not enough data in Server Hello")
	}
	helloVersion := s.readInt(2)
	s.moveForwardsBy(32)
	sessionIDLength := s.readInt(1)
	s.moveForwardsBy(sessionIDLength)

	cipherSuite := s.readInt(2)
	s.moveForwardsBy(1) // compression method

	extensionsLength := s.readInt(2)
	es := newByteStream(s.getBytes(extensionsLength))
	var exts []string
	for es.hasMore() {
		typ := es.readInt(2)
		extLength := es.readInt(2)
		es.moveForwardsBy(extLength)
		exts = append(exts, strconv.Itoa(typ))
	}

	ja3sStr := strings.Join([]string{
		strconv.Itoa(helloVersion),
		strconv.Itoa(cipherSuite),
		strings.Join(exts, "-"),
	}, ",")
	ja3sHash := fmt.Sprintf("%x", md5.Sum([]byte(ja3sStr))) // #nosec G401 -- MD5/SHA1 is an intentional CyberChef operation, not a security control

	switch outputFormat {
	case "JA3S string":
		return core.NewDish([]byte(ja3sStr), core.TypeString), nil
	case "Full details":
		out := fmt.Sprintf("Hash digest:\n%s\n\nFull JA3S string:\n%s\n\nTLS Version:\n%d\nCipher Suite:\n%d\nExtensions:\n%s",
			ja3sHash, ja3sStr, helloVersion, cipherSuite, strings.Join(exts, "-"))
		return core.NewDish([]byte(out), core.TypeString), nil
	default: // Hash digest
		return core.NewDish([]byte(ja3sHash), core.TypeString), nil
	}
}
