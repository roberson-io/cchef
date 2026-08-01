package ops

import (
	"crypto/md5" // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(HASSHClientFingerprint{})
}

// HASSHClientFingerprint generates a HASSH fingerprint from an SSH client
// Key Exchange Init message.
type HASSHClientFingerprint struct{}

// Meta returns the operation metadata.
func (HASSHClientFingerprint) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "HASSH Client Fingerprint",
		Module:      "Crypto",
		Description: "Generates a HASSH fingerprint to help identify SSH clients based on hashing together values from the Client Key Exchange Init message. Input: a hex stream of the SSH_MSG_KEXINIT packet application layer from Client to Server.",
		InfoURL:     "https://engineering.salesforce.com/open-sourcing-hassh-abed3ae5044c",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HASSHClientFingerprint) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Hash digest", "HASSH algorithms string", "Full details"}},
	}
}

// Run generates the HASSH fingerprint. Ported from CyberChef HASSHClientFingerprint.mjs.
func (HASSHClientFingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)
	outputFormat := args[1].(string)

	data := fingerprintBytes(in.String(), inputFormat)
	s := newByteStream(data)

	length := s.readInt(4)
	if s.length() != length+4 {
		return nil, fingerprintError("incorrect packet length")
	}
	paddingLength := s.readInt(1)
	if s.readInt(1) != 20 {
		return nil, fingerprintError("not a Key Exchange Init")
	}
	s.moveForwardsBy(16) // cookie

	kexAlgos := s.readString(s.readInt(4))
	s.moveForwardsBy(s.readInt(4)) // server host key algos

	encAlgosC2S := s.readString(s.readInt(4))
	s.moveForwardsBy(s.readInt(4)) // enc algos S2C

	macAlgosC2S := s.readString(s.readInt(4))
	s.moveForwardsBy(s.readInt(4)) // mac algos S2C

	compAlgosC2S := s.readString(s.readInt(4))
	s.moveForwardsBy(s.readInt(4)) // comp algos S2C

	s.moveForwardsBy(s.readInt(4)) // langs C2S
	s.moveForwardsBy(s.readInt(4)) // langs S2C
	s.moveForwardsBy(1)            // first_kex_packet_follows
	s.moveForwardsBy(4)            // reserved
	s.moveForwardsBy(paddingLength)

	hasshStr := strings.Join([]string{kexAlgos, encAlgosC2S, macAlgosC2S, compAlgosC2S}, ";")
	hasshHash := fmt.Sprintf("%x", md5.Sum([]byte(hasshStr))) // #nosec G401 -- MD5/SHA1 is an intentional CyberChef operation, not a security control

	switch outputFormat {
	case "HASSH algorithms string":
		return core.NewDish([]byte(hasshStr), core.TypeString), nil
	case "Full details":
		out := fmt.Sprintf("Hash digest:\n%s\n\nFull HASSH algorithms string:\n%s\n\nKey Exchange Algorithms:\n%s\nEncryption Algorithms Client to Server:\n%s\nMAC Algorithms Client to Server:\n%s\nCompression Algorithms Client to Server:\n%s",
			hasshHash, hasshStr, kexAlgos, encAlgosC2S, macAlgosC2S, compAlgosC2S)
		return core.NewDish([]byte(out), core.TypeString), nil
	default: // Hash digest
		return core.NewDish([]byte(hasshHash), core.TypeString), nil
	}
}
