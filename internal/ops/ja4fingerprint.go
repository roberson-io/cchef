package ops

import (
	"fmt"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(JA4Fingerprint{})
}

// JA4Fingerprint generates a JA4 fingerprint from a TLS Client Hello.
type JA4Fingerprint struct{}

// Meta returns the operation metadata.
func (JA4Fingerprint) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "JA4 Fingerprint",
		Module:      "Crypto",
		Description: "Generates a JA4 fingerprint to help identify TLS clients based on hashing together values from the Client Hello. Input: a hex stream of the TLS or QUIC Client Hello packet application layer.",
		InfoURL:     "https://medium.com/foxio/ja4-network-fingerprinting-9376fe9ca637",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (JA4Fingerprint) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"JA4", "JA4 Original Rendering", "JA4 Raw", "JA4 Raw Original Rendering", "All"}},
	}
}

// Run generates the JA4 fingerprint. Ported from CyberChef JA4Fingerprint.mjs.
func (JA4Fingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data, err := fingerprintBytes(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	ja4, err := toJA4(data)
	if err != nil {
		return nil, err
	}
	var out string
	switch args[1].(string) {
	case "JA4 Original Rendering":
		out = ja4["JA4_o"]
	case "JA4 Raw":
		out = ja4["JA4_r"]
	case "JA4 Raw Original Rendering":
		out = ja4["JA4_ro"]
	case "All":
		out = fmt.Sprintf("JA4:    %s\nJA4_o:  %s\nJA4_r:  %s\nJA4_ro: %s", ja4["JA4"], ja4["JA4_o"], ja4["JA4_r"], ja4["JA4_ro"])
	default: // JA4
		out = ja4["JA4"]
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
