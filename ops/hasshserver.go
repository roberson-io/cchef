package ops

import (
	"crypto/md5" // #nosec G501 -- crypto/md5 required by the ported CyberChef operation, not a security control
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
)

func init() {
	core.Register(HASSHServerFingerprint{})
}

// HASSHServerFingerprint generates a HASSH fingerprint from an SSH server
// Key Exchange Init message.
type HASSHServerFingerprint struct{}

// Meta returns the operation metadata.
func (HASSHServerFingerprint) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "HASSH Server Fingerprint",
		Module:      "Crypto",
		Description: "Generates a HASSH fingerprint to help identify SSH servers based on hashing together values from the Server Key Exchange Init message. Input: a hex stream of the SSH_MSG_KEXINIT packet application layer from Server to Client.",
		InfoURL:     "https://engineering.salesforce.com/open-sourcing-hassh-abed3ae5044c",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HASSHServerFingerprint) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Input format", Type: core.ArgOption, Value: []string{"Hex", "Base64", "Raw"}},
		{Name: "Output format", Type: core.ArgOption, Value: []string{"Hash digest", "HASSH algorithms string", "Full details"}},
	}
}

// Run generates the HASSH server fingerprint.
func (HASSHServerFingerprint) Run(in *core.Dish, args []any) (*core.Dish, error) {
	inputFormat := args[0].(string)
	outputFormat := args[1].(string)

	data := fingerprintBytes(in.String(), inputFormat)
	s := bytestream.New(data)

	length := s.ReadInt(4)
	if s.Length() != length+4 {
		return nil, fingerprintError("incorrect packet length")
	}
	paddingLength := s.ReadInt(1)
	if s.ReadInt(1) != 20 {
		return nil, fingerprintError("not a Key Exchange Init")
	}
	s.MoveForwardsBy(16) // cookie

	kexAlgos := s.ReadString(s.ReadInt(4))
	s.MoveForwardsBy(s.ReadInt(4)) // server host key algos

	s.MoveForwardsBy(s.ReadInt(4)) // enc algos C2S
	encAlgosS2C := s.ReadString(s.ReadInt(4))

	s.MoveForwardsBy(s.ReadInt(4)) // mac algos C2S
	macAlgosS2C := s.ReadString(s.ReadInt(4))

	s.MoveForwardsBy(s.ReadInt(4)) // comp algos C2S
	compAlgosS2C := s.ReadString(s.ReadInt(4))

	s.MoveForwardsBy(s.ReadInt(4)) // langs C2S
	s.MoveForwardsBy(s.ReadInt(4)) // langs S2C
	s.MoveForwardsBy(1)            // first_kex_packet_follows
	s.MoveForwardsBy(4)            // reserved
	s.MoveForwardsBy(paddingLength)

	hasshStr := strings.Join([]string{kexAlgos, encAlgosS2C, macAlgosS2C, compAlgosS2C}, ";")
	hasshHash := fmt.Sprintf("%x", md5.Sum([]byte(hasshStr))) // #nosec G401 -- MD5/SHA1 is an intentional CyberChef operation, not a security control

	switch outputFormat {
	case "HASSH algorithms string":
		return core.NewDish([]byte(hasshStr), core.TypeString), nil
	case "Full details":
		out := fmt.Sprintf("Hash digest:\n%s\n\nFull HASSH algorithms string:\n%s\n\nKey Exchange Algorithms:\n%s\nEncryption Algorithms Server to Client:\n%s\nMAC Algorithms Server to Client:\n%s\nCompression Algorithms Server to Client:\n%s",
			hasshHash, hasshStr, kexAlgos, encAlgosS2C, macAlgosS2C, compAlgosS2C)
		return core.NewDish([]byte(out), core.TypeString), nil
	default: // Hash digest
		return core.NewDish([]byte(hasshHash), core.TypeString), nil
	}
}
