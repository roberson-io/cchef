package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JA4ServerFingerprint{})
}

// JA4ServerFingerprint generates a JA4S fingerprint from a TLS Server Hello.
type JA4ServerFingerprint struct{}

// Meta returns the operation metadata.
func (JA4ServerFingerprint) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JA4Server Fingerprint",
		Module:      "Crypto",
		Description: "Generates a JA4Server Fingerprint (JA4S) to help identify TLS servers or sessions based on hashing together values from the Server Hello. Input: a hex stream of the TLS or QUIC Server Hello packet application layer.",
		InfoURL:     "https://medium.com/foxio/ja4-network-fingerprinting-9376fe9ca637",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JA4ServerFingerprint) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"JA4S", "JA4S Raw", "Both"}},
	}
}

// Run generates the JA4S fingerprint. Ported from CyberChef JA4ServerFingerprint.mjs.
func (JA4ServerFingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data, err := fingerprintBytes(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	ja4s, err := toJA4S(data)
	if err != nil {
		return nil, err
	}
	var out string
	switch args[1].(string) {
	case "JA4S Raw":
		out = ja4s["JA4S_r"]
	case "Both":
		out = fmt.Sprintf("JA4S:   %s\nJA4S_r: %s", ja4s["JA4S"], ja4s["JA4S_r"])
	default: // JA4S
		out = ja4s["JA4S"]
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
