package ops

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseSSHHostKey{})
}

var (
	sshKeyPrefixRe = regexp.MustCompile(`^(?:ssh|ecdsa-sha2)\S+\s+(\S*)`)
	sshHexRe       = regexp.MustCompile(`^(?:[\dA-Fa-f]{2}[ ,;:]?)+$`)
	sshB64Re       = regexp.MustCompile(`^\s*(?:[A-Za-z\d+/]{4})+(?:[A-Za-z\d+/]{2}==|[A-Za-z\d+/]{3}=)?\s*$`)
)

// fromBase64Std decodes standard base64, padding the input if needed.
func fromBase64Std(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if m := len(s) % 4; m != 0 {
		s += strings.Repeat("=", 4-m)
	}
	return base64.StdEncoding.DecodeString(s)
}

// sshConvertKeyToBinary extracts the key body and decodes it per the format.
func sshConvertKeyToBinary(inputKey, format string) ([]byte, error) {
	if m := sshKeyPrefixRe.FindStringSubmatch(inputKey); m != nil {
		inputKey = m[1]
	}
	if format == "Auto" {
		switch {
		case sshHexRe.MatchString(inputKey):
			format = "Hex"
		case sshB64Re.MatchString(inputKey):
			format = "Base64"
		default:
			return nil, fmt.Errorf("unable to detect input key format")
		}
	}
	if format == "Hex" {
		return hexToBytes(inputKey), nil
	}
	return fromBase64Std(inputKey)
}

// sshParseKey reads the key's length-prefixed fields, returning each as hex.
func sshParseKey(key []byte) []string {
	var fields []string
	for len(key) > 0 {
		if len(key) < 4 {
			break
		}
		decodedLength := int(key[0])<<24 | int(key[1])<<16 | int(key[2])<<8 | int(key[3])
		if decodedLength <= 0 {
			break
		}
		end := min(4+decodedLength, len(key))
		fields = append(fields, hex.EncodeToString(key[4:end]))
		key = key[end:]
	}
	return fields
}

// ParseSSHHostKey parses an SSH host key into its type and parameters.
type ParseSSHHostKey struct{}

// Meta returns the operation metadata.
func (ParseSSHHostKey) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse SSH Host Key",
		Module:      "Default",
		Description: "Parses a SSH host key and extracts fields from it.<br><br>The key type and fields are parsed. Supported key types are RSA, DSA, ECDSA and Ed25519.",
		InfoURL:     "https://wikipedia.org/wiki/Secure_Shell",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseSSHHostKey) Args() []core.ArgDef {
	return []core.ArgDef{{Name: "Input Format", Type: core.ArgOption, Value: []string{"Auto", "Base64", "Hex"}}}
}

// Run parses the key. Ported from CyberChef ParseSSHHostKey.mjs.
func (ParseSSHHostKey) Run(in *core.Dish, args []any) (*core.Dish, error) {
	key, err := sshConvertKeyToBinary(strings.TrimSpace(in.String()), args[0].(string))
	if err != nil {
		return nil, err
	}
	fields := sshParseKey(key)
	if len(fields) == 0 {
		return nil, fmt.Errorf("unable to parse key")
	}
	keyType := string(hexToBytes(fields[0]))

	var b strings.Builder
	fmt.Fprintf(&b, "Key type: %s", keyType)
	switch {
	case keyType == "ssh-rsa":
		fmt.Fprintf(&b, "\nExponent: 0x%s\nModulus: 0x%s", fields[1], fields[2])
	case keyType == "ssh-dss":
		fmt.Fprintf(&b, "\np: 0x%s\nq: 0x%s\ng: 0x%s\ny: 0x%s", fields[1], fields[2], fields[3], fields[4])
	case strings.HasPrefix(keyType, "ecdsa-sha2"):
		fmt.Fprintf(&b, "\nCurve: %s\nPoint: 0x%s", string(hexToBytes(fields[1])), strings.Join(fields[2:], ","))
	case keyType == "ssh-ed25519":
		fmt.Fprintf(&b, "\nx: 0x%s", fields[1])
	default:
		fmt.Fprintf(&b, "\nUnsupported key type.\nParameters: %s", strings.Join(fields[1:], ","))
	}
	return core.NewDish([]byte(b.String()), core.TypeString), nil
}
