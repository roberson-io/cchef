package ops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(GenerateAllHashes{})
}

// genHashEntry describes one hash in the generated report: its display name, the
// registry operation, its raw arguments, and whether it consumes the input as a
// (Latin-1) string rather than the raw byte buffer.
type genHashEntry struct {
	name string
	op   string
	str  bool
	args []any
}

// blake2Params builds the [size, "Hex", empty-key] argument list the BLAKE2
// operations take.
func blake2Params(size string) []any {
	return []any{size, "Hex", core.ToggleString{Value: "", Option: "UTF8"}}
}

// genAllHashesList mirrors the hash table in CyberChef's GenerateAllHashes, in
// the same order (which determines the output order).
var genAllHashesList = []genHashEntry{
	{"MD2", "MD2", false, nil},
	{"MD4", "MD4", false, nil},
	{"MD5", "MD5", false, nil},
	{"MD6", "MD6", true, nil},
	{"SHA0", "SHA0", false, nil},
	{"SHA1", "SHA1", false, nil},
	{"SHA2 224", "SHA224", false, nil},
	{"SHA2 256", "SHA256", false, nil},
	{"SHA2 384", "SHA384", false, nil},
	{"SHA2 512", "SHA512", false, nil},
	{"SHA3 224", "SHA3", false, []any{"224"}},
	{"SHA3 256", "SHA3", false, []any{"256"}},
	{"SHA3 384", "SHA3", false, []any{"384"}},
	{"SHA3 512", "SHA3", false, []any{"512"}},
	{"Keccak 224", "Keccak", false, []any{"224"}},
	{"Keccak 256", "Keccak", false, []any{"256"}},
	{"Keccak 384", "Keccak", false, []any{"384"}},
	{"Keccak 512", "Keccak", false, []any{"512"}},
	{"Shake 128", "Shake", false, []any{"128", 256}},
	{"Shake 256", "Shake", false, []any{"256", 512}},
	{"RIPEMD-128", "RIPEMD", false, []any{"128"}},
	{"RIPEMD-160", "RIPEMD", false, []any{"160"}},
	{"RIPEMD-256", "RIPEMD", false, []any{"256"}},
	{"RIPEMD-320", "RIPEMD", false, []any{"320"}},
	{"HAS-160", "HAS-160", false, nil},
	{"Whirlpool-0", "Whirlpool", false, []any{"Whirlpool-0"}},
	{"Whirlpool-T", "Whirlpool", false, []any{"Whirlpool-T"}},
	{"Whirlpool", "Whirlpool", false, []any{"Whirlpool"}},
	{"BLAKE2b-128", "BLAKE2b", false, blake2Params("128")},
	{"BLAKE2b-160", "BLAKE2b", false, blake2Params("160")},
	{"BLAKE2b-256", "BLAKE2b", false, blake2Params("256")},
	{"BLAKE2b-384", "BLAKE2b", false, blake2Params("384")},
	{"BLAKE2b-512", "BLAKE2b", false, blake2Params("512")},
	{"BLAKE2s-128", "BLAKE2s", false, blake2Params("128")},
	{"BLAKE2s-160", "BLAKE2s", false, blake2Params("160")},
	{"BLAKE2s-256", "BLAKE2s", false, blake2Params("256")},
	{"Streebog-256", "Streebog", false, []any{"256"}},
	{"Streebog-512", "Streebog", false, []any{"512"}},
	{"GOST", "GOST Hash", false, []any{"GOST 28147 (1994)", "256", "D-A"}},
	{"LM Hash", "LM Hash", true, nil},
	{"NT Hash", "NT Hash", true, nil},
	{"SSDEEP", "SSDEEP", true, nil},
	{"CTPH", "CTPH", true, nil},
}

// genHashNameWidth is the column the digests are aligned to (the longest name
// plus its colon).
const genHashNameWidth = 13

// GenerateAllHashes runs every available hash and checksum over the input.
// Ported from CyberChef GenerateAllHashes.mjs; it composes the individual hash
// operations rather than reimplementing them.
type GenerateAllHashes struct{}

// Meta returns the operation metadata.
func (GenerateAllHashes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate all hashes",
		Module:      "Crypto",
		Description: "Generates all available hashes and checksums for the input.",
		InfoURL:     "https://wikipedia.org/wiki/Comparison_of_cryptographic_hash_functions",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateAllHashes) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Length (bits)", Type: core.ArgOption, Value: []string{"All", "128", "160", "224", "256", "320", "384", "512"}},
		{Name: "Include names", Type: core.ArgBoolean, Value: true},
	}
}

// Run generates all hashes.
func (GenerateAllHashes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	length := args[0].(string)
	includeNames := args[1].(bool)

	bufDish := core.NewDish(in.Bytes(), core.TypeArrayBuffer)
	// str-input hashes receive the Latin-1 decode of the bytes, as CyberChef's
	// arrayBufferToStr(input, false) produces.
	strDish := core.NewDish([]byte(byteArrayToChars(in.Bytes())), core.TypeString)

	var out strings.Builder
	for _, e := range genAllHashesList {
		dish := bufDish
		if e.str {
			dish = strDish
		}
		res, err := core.Recipe{{Op: e.op, Args: e.args}}.Execute(dish)
		if err != nil {
			return nil, err
		}
		out.WriteString(genFormatDigest(res.String(), length, includeNames, e.name))
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// genFormatDigest renders one digest line, filtering by requested bit-length and
// optionally prefixing the aligned name.
func genFormatDigest(digest, length string, includeNames bool, name string) string {
	if length != "All" {
		want, _ := strconv.Atoi(length)
		if len(digest)*4 != want {
			return ""
		}
	}
	if !includeNames {
		return digest + "\n"
	}
	return fmt.Sprintf("%s:%s%s\n", name, strings.Repeat(" ", genHashNameWidth-len(name)), digest)
}
