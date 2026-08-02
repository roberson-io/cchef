package ops

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// uuidLayout is what the operation will read. Only the versions the standard
// defines are accepted, and only the four variant digits that go with them; the
// two UUIDs reserved by name are let through as themselves.
var uuidLayout = regexp.MustCompile(`^(?i:` +
	`[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}` +
	`|00000000-0000-0000-0000-000000000000` +
	`|ffffffff-ffff-ffff-ffff-ffffffffffff)$`)

// errInvalidUUID is what every unreadable input gets, whichever part of the
// layout it failed on.
//
//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
var errInvalidUUID = errors.New("Invalid UUID")

// The count a version 1 or 6 timestamp keeps, and where it starts from: tenths
// of a microsecond since the Gregorian calendar began, which is a long way
// before the Unix epoch.
const (
	uuidTicksPerMillisecond = 10000
	uuidGregorianToUnix     = 122192928000000000
)

// uuidVersionDigit is where the version sits in the written form.
const uuidVersionDigit = 14

// AnalyseUUID reads the version and any embedded metadata back out of a UUID.
type AnalyseUUID struct{}

// Meta returns the operation metadata.
func (AnalyseUUID) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Analyse UUID",
		Module: "Default",
		Description: "Reads a UUID and reports what it carries: the version, the " +
			"128-bit value itself, and for the versions that hold one, the timestamp " +
			"it was made at.<br><br>Versions 1, 6 and 7 embed a timestamp; version 1 " +
			"and 6 also carry a node identifier and clock sequence, and version 7 " +
			"carries two stretches of randomness. The other versions hold nothing to " +
			"read back.",
		InfoURL:    "https://wikipedia.org/wiki/Universally_unique_identifier",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (AnalyseUUID) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Include Metadata", Type: core.ArgBoolean, Value: true},
	}
}

// Run analyses the UUID.
func (AnalyseUUID) Run(in *core.Dish, args []any) (*core.Dish, error) {
	text := jsTrimSpace(in.String())
	if !uuidLayout.MatchString(text) {
		return nil, errInvalidUUID
	}
	// The layout has already been checked, so the digits are all hex and there
	// are exactly the right number of them.
	raw, _ := hex.DecodeString(strings.ReplaceAll(text, "-", ""))
	version, _ := hexNibble(rune(text[uuidVersionDigit]))

	sections := []string{"Version:\n" + strconv.Itoa(int(version))}
	if includeMetadata, _ := args[0].(bool); includeMetadata {
		sections = append(sections, uuidMetadata(int(version), raw))
	}
	sections = append(sections, "UUID Integer:\n"+new(big.Int).SetBytes(raw).String())

	return core.NewDish([]byte(strings.Join(sections, "\n\n")), core.TypeString), nil
}

// uuidMetadata describes what the UUID holds beyond its version, for the three
// versions that hold anything.
func uuidMetadata(version int, raw []byte) string {
	switch version {
	case 1, 6:
		return uuidTimeFields(raw)
	case 7:
		return uuidUnixTimeFields(raw)
	default:
		return "No metadata available. Only versions 1, 6, 7 are supported."
	}
}

// uuidTimeFields reads a version 1 or 6 UUID. The two lay the same timestamp out
// differently: version 1 puts its low bits first, version 6 its high bits, so
// that sorting the written form sorts by time.
func uuidTimeFields(raw []byte) string {
	var ticks int64
	if raw[6]>>4 == 1 {
		ticks = int64(be16(raw[6:])&0x0fff)<<48 |
			int64(be16(raw[4:]))<<32 |
			int64(be32(raw[0:]))
	} else {
		ticks = int64(be32(raw[0:]))<<28 |
			int64(be16(raw[4:]))<<12 |
			int64(be16(raw[6:])&0x0fff)
	}
	milliseconds := (ticks - uuidGregorianToUnix) / uuidTicksPerMillisecond

	return uuidFields(
		[2]string{"Timestamp", strconv.FormatInt(milliseconds, 10)},
		[2]string{"Timestamp (ISO)", jsISOTimestamp(milliseconds)},
		[2]string{"Node", uuidHex(raw[10:], ":")},
		[2]string{"Clock", strconv.Itoa(int(raw[8]&0x3f)<<8 | int(raw[9]))},
	)
}

// uuidUnixTimeFields reads a version 7 UUID, whose timestamp is already a count
// of milliseconds since the Unix epoch and whose remaining bits are random.
func uuidUnixTimeFields(raw []byte) string {
	milliseconds := int64(be32(raw[0:]))<<16 | int64(be16(raw[4:]))

	return uuidFields(
		[2]string{"Timestamp", strconv.FormatInt(milliseconds, 10)},
		[2]string{"Timestamp (ISO)", jsISOTimestamp(milliseconds)},
		[2]string{"Rand A", strconv.Itoa(int(raw[6]&0x0f)<<8 | int(raw[7]))},
		[2]string{"Rand B", uuidHex(raw[8:], "")},
	)
}

// uuidFields lays each label out with its value, one to a paragraph.
func uuidFields(fields ...[2]string) string {
	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = field[0] + ":\n" + field[1]
	}
	return strings.Join(out, "\n\n")
}

// uuidHex writes bytes as upper case hex, separated as asked.
func uuidHex(b []byte, sep string) string {
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = fmt.Sprintf("%02X", v)
	}
	return strings.Join(parts, sep)
}

// be16 reads the big-endian sixteen-bit integers the layout is described in.
func be16(b []byte) uint16 { return uint16(b[0])<<8 | uint16(b[1]) }

// be32 reads the big-endian thirty-two-bit ones.
func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func init() { core.Register(AnalyseUUID{}) }
