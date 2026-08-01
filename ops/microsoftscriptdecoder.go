package ops

import (
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(MicrosoftScriptDecoder{})
}

// MicrosoftScriptDecoder struct.
type MicrosoftScriptDecoder struct{}

// Meta returns the operation metadata.
func (MicrosoftScriptDecoder) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Microsoft Script Decoder",
		Module:      "Default",
		Description: "Decodes Microsoft Encoded Script files that have been encoded with Microsoft's custom encoding.",
		InfoURL:     "https://wikipedia.org/wiki/JScript.Encode",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (MicrosoftScriptDecoder) Args() []core.ArgDef { return nil }

var (
	reMSScript    = regexp.MustCompile(`#@~\^.{6}==(.+).{6}==\^#~@`)
	msPreReplacer = strings.NewReplacer("@&", "\n", "@#", "\r", "@*", ">", "@!", "<", "@$", "@")
)

// Run extracts and decodes an encoded Microsoft script; non-matching input yields
// an empty string.
func (MicrosoftScriptDecoder) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	m := reMSScript.FindStringSubmatch(in.String())
	if m == nil {
		return core.NewDish([]byte(""), core.TypeString), nil
	}
	return core.NewDish([]byte(msDecode(m[1])), core.TypeString), nil
}

// msDecode reverses the Microsoft Script Encoder substitution. It iterates bytes
// (equivalent to the original's UTF-16 code-unit iteration for this algorithm,
// since only bytes 9 and 32..127 are transformed).
func msDecode(data string) string {
	data = msPreReplacer.Replace(data)
	var b strings.Builder
	index := -1
	for i := 0; i < len(data); i++ {
		c := data[i]
		if c < 128 {
			index++
		}
		if (c == 9 || (c > 31 && c < 128)) && c != 60 && c != 62 && c != 64 {
			b.WriteByte(dDecode[c][dCombination[index%64]])
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

var dDecode = [128]string{
	"",
	"",
	"",
	"",
	"",
	"",
	"",
	"",
	"",
	"\x57\x6e\x7b",
	"\x4a\x4c\x41",
	"\x0b\x0b\x0b",
	"\x0c\x0c\x0c",
	"\x4a\x4c\x41",
	"\x0e\x0e\x0e",
	"\x0f\x0f\x0f",
	"\x10\x10\x10",
	"\x11\x11\x11",
	"\x12\x12\x12",
	"\x13\x13\x13",
	"\x14\x14\x14",
	"\x15\x15\x15",
	"\x16\x16\x16",
	"\x17\x17\x17",
	"\x18\x18\x18",
	"\x19\x19\x19",
	"\x1a\x1a\x1a",
	"\x1b\x1b\x1b",
	"\x1c\x1c\x1c",
	"\x1d\x1d\x1d",
	"\x1e\x1e\x1e",
	"\x1f\x1f\x1f",
	"\x2e\x2d\x32",
	"\x47\x75\x30",
	"\x7a\x52\x21",
	"\x56\x60\x29",
	"\x42\x71\x5b",
	"\x6a\x5e\x38",
	"\x2f\x49\x33",
	"\x26\x5c\x3d",
	"\x49\x62\x58",
	"\x41\x7d\x3a",
	"\x34\x29\x35",
	"\x32\x36\x65",
	"\x5b\x20\x39",
	"\x76\x7c\x5c",
	"\x72\x7a\x56",
	"\x43\x7f\x73",
	"\x38\x6b\x66",
	"\x39\x63\x4e",
	"\x70\x33\x45",
	"\x45\x2b\x6b",
	"\x68\x68\x62",
	"\x71\x51\x59",
	"\x4f\x66\x78",
	"\x09\x76\x5e",
	"\x62\x31\x7d",
	"\x44\x64\x4a",
	"\x23\x54\x6d",
	"\x75\x43\x71",
	"\x4a\x4c\x41",
	"\x7e\x3a\x60",
	"\x4a\x4c\x41",
	"\x5e\x7e\x53",
	"\x40\x4c\x40",
	"\x77\x45\x42",
	"\x4a\x2c\x27",
	"\x61\x2a\x48",
	"\x5d\x74\x72",
	"\x22\x27\x75",
	"\x4b\x37\x31",
	"\x6f\x44\x37",
	"\x4e\x79\x4d",
	"\x3b\x59\x52",
	"\x4c\x2f\x22",
	"\x50\x6f\x54",
	"\x67\x26\x6a",
	"\x2a\x72\x47",
	"\x7d\x6a\x64",
	"\x74\x39\x2d",
	"\x54\x7b\x20",
	"\x2b\x3f\x7f",
	"\x2d\x38\x2e",
	"\x2c\x77\x4c",
	"\x30\x67\x5d",
	"\x6e\x53\x7e",
	"\x6b\x47\x6c",
	"\x66\x34\x6f",
	"\x35\x78\x79",
	"\x25\x5d\x74",
	"\x21\x30\x43",
	"\x64\x23\x26",
	"\x4d\x5a\x76",
	"\x52\x5b\x25",
	"\x63\x6c\x24",
	"\x3f\x48\x2b",
	"\x7b\x55\x28",
	"\x78\x70\x23",
	"\x29\x69\x41",
	"\x28\x2e\x34",
	"\x73\x4c\x09",
	"\x59\x21\x2a",
	"\x33\x24\x44",
	"\x7f\x4e\x3f",
	"\x6d\x50\x77",
	"\x55\x09\x3b",
	"\x53\x56\x55",
	"\x7c\x73\x69",
	"\x3a\x35\x61",
	"\x5f\x61\x63",
	"\x65\x4b\x50",
	"\x46\x58\x67",
	"\x58\x3b\x51",
	"\x31\x57\x49",
	"\x69\x22\x4f",
	"\x6c\x6d\x46",
	"\x5a\x4d\x68",
	"\x48\x25\x7c",
	"\x27\x28\x36",
	"\x5c\x46\x70",
	"\x3d\x4a\x6e",
	"\x24\x32\x7a",
	"\x79\x41\x2f",
	"\x37\x3d\x5f",
	"\x60\x5f\x4b",
	"\x51\x4f\x5a",
	"\x20\x42\x2c",
	"\x36\x65\x57",
}

var dCombination = [64]int{
	0, 1, 2, 0, 1, 2, 1, 2, 2, 1, 2, 1, 0, 2, 1, 2, 0, 2, 1, 2, 0, 0, 1, 2, 2, 1, 0, 2, 1, 2, 2, 1, 0, 0, 2, 1, 2, 1, 2, 0, 2, 0, 0, 1, 2, 0, 2, 1, 0, 2, 1, 2, 0, 0, 1, 2, 2, 0, 0, 1, 2, 0, 2, 1,
}
