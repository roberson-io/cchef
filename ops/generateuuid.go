package ops

import (
	"crypto/md5" // #nosec G501 -- version 3 UUIDs are defined in terms of MD5
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- version 5 UUIDs are defined in terms of SHA-1
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/roberson-io/cchef/core"
)

// The UUID versions the operation offers, and where the default sits.
var uuidVersions = []string{"v1", "v3", "v4", "v5", "v6", "v7"}

const uuidDefaultVersion = 2 // v4, which is drawn purely at random

// uuidDefaultNamespace is the namespace versions 3 and 5 hash under unless
// another is given.
const uuidDefaultNamespace = "1b671a64-40d5-491e-99b0-da01ff1f3341"

// uuidSize is how many bytes a UUID holds.
const uuidSize = 16

// The bits that mark a UUID as following the standard, and where the version
// digit sits within the bytes.
const (
	uuidVariantByte = 8
	uuidVersionByte = 6
)

// uuidNanosPerMillisecond is how many hundred-nanosecond ticks fit in the
// millisecond a version 1 timestamp counts in.
const uuidNanosPerMillisecond = 10000

// uuidDrawClockseq asks uuidV1Bytes to take a clock sequence from the
// randomness it was given rather than being handed one.
const uuidDrawClockseq = -1

// uuidMaxNanos is where the sub-millisecond counter wraps, taking a fresh node
// with it so two UUIDs made in the same millisecond can never collide.
const uuidMaxNanos = 10000

// GenerateUUID makes a universally unique identifier.
type GenerateUUID struct{}

// Meta returns the operation metadata.
func (GenerateUUID) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Generate UUID",
		Module: "Default",
		Description: "Generates an RFC 9562 (formerly RFC 4122) Universally Unique " +
			"Identifier (UUID), also known as a Globally Unique Identifier (GUID)." +
			"<br><br>The versions on offer are:<br><ul>" +
			"<li><strong>v1</strong>: Timestamp-based</li>" +
			"<li><strong>v3</strong>: Namespace w/ MD5</li>" +
			"<li><strong>v4</strong>: Random (default)</li>" +
			"<li><strong>v5</strong>: Namespace w/ SHA-1</li>" +
			"<li><strong>v6</strong>: Timestamp, reordered</li>" +
			"<li><strong>v7</strong>: Unix Epoch time-based</li></ul>" +
			"Versions 3 and 5 hash the input under the namespace given, so the same " +
			"pair always gives the same UUID. The rest ignore the input.",
		InfoURL:    "https://wikipedia.org/wiki/Universally_unique_identifier",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateUUID) Args() []core.ArgDef {
	return []core.ArgDef{
		{
			Name:         "Version",
			Type:         core.ArgOption,
			Value:        uuidVersions,
			DefaultIndex: uuidDefaultVersion,
		},
		{Name: "Namespace", Type: core.ArgString, Value: uuidDefaultNamespace},
	}
}

// Run generates the UUID.
func (GenerateUUID) Run(in *core.Dish, args []any) (*core.Dish, error) {
	version, _ := args[0].(string)
	namespace, _ := args[1].(string)

	id, err := generateUUID(version, namespace, in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(id), core.TypeString), nil
}

// generateUUID makes one UUID of the version asked for. Only versions 3 and 5
// look at the name or the namespace.
func generateUUID(version, namespace string, name []byte) (string, error) {
	if version == "v3" || version == "v5" {
		return uuidNameBased(version, namespace, name)
	}

	random, err := uuidRandom()
	if err != nil {
		return "", err
	}
	now := uuidNow()

	switch version {
	case "v1":
		return uuidStringify(uuidState.nextV1(now, random)), nil
	case "v4":
		return uuidStringify(uuidV4Bytes(random)), nil
	case "v6":
		return uuidStringify(uuidV1ToV6(uuidState.nextV6(now, random))), nil
	case "v7":
		return uuidStringify(uuidState.nextV7(now, random)), nil
	default:
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return "", errors.New("Invalid UUID version")
	}
}

// uuidNameBased hashes the name under the namespace, which gives the same UUID
// for the same pair every time.
func uuidNameBased(version, namespace string, name []byte) (string, error) {
	if !uuidLayout.MatchString(namespace) {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return "", errors.New("Invalid UUID namespace")
	}
	prefix, _ := hex.DecodeString(strings.ReplaceAll(namespace, "-", ""))

	var sum []byte
	var mark byte
	if version == "v3" {
		digest := md5.Sum(append(prefix, name...)) // #nosec G401 -- the standard defines version 3 this way
		sum, mark = digest[:], 0x30
	} else {
		digest := sha1.Sum(append(prefix, name...)) // #nosec G401 -- the standard defines version 5 this way
		sum, mark = digest[:], 0x50
	}

	out := make([]byte, uuidSize)
	copy(out, sum)
	uuidMark(out, mark)
	return uuidStringify(out), nil
}

// uuidMark writes the version and variant bits that say which kind of UUID this
// is and that it follows the standard.
func uuidMark(b []byte, version byte) {
	b[uuidVersionByte] = b[uuidVersionByte]&0x0f | version
	b[uuidVariantByte] = b[uuidVariantByte]&0x3f | 0x80
}

// uuidV4Bytes lays out a UUID that is random throughout but for its markings.
func uuidV4Bytes(random []byte) []byte {
	out := make([]byte, uuidSize)
	copy(out, random)
	uuidMark(out, 0x40)
	return out
}

// uuidV1Bytes lays out a version 1 UUID: a count of hundred-nanosecond ticks
// since the Gregorian calendar began, split across three fields, followed by a
// clock sequence and a node.
//
// The timestamp arithmetic is transliterated from
// CyberChef's node_modules/uuid/dist/v1.js, which works the high half out by
// dividing before it multiplies. Doing that in floating point is what the
// package does and what the bytes must match.
func uuidV1Bytes(random []byte, msecs int64, nsecs, clockseq int, node []byte) []byte {
	out := make([]byte, uuidSize)
	if clockseq < 0 {
		clockseq = (int(random[8])<<8 | int(random[9])) & 0x3fff
	}
	if node == nil {
		node = random[10:16]
	}

	msecs += uuidGregorianToUnix / uuidNanosPerMillisecond

	low := ((msecs&0xfffffff)*uuidNanosPerMillisecond + int64(nsecs)) % 0x100000000 // #nosec G115 -- narrowed to the width the layout gives the field
	out[0] = byte(low >> 24)                                                        // #nosec G115 -- narrowed to the width the layout gives the field
	out[1] = byte(low >> 16)                                                        // #nosec G115 -- narrowed to the width the layout gives the field
	out[2] = byte(low >> 8)                                                         // #nosec G115 -- narrowed to the width the layout gives the field
	out[3] = byte(low)                                                              // #nosec G115 -- narrowed to the width the layout gives the field

	high := jsToInt32(float64(msecs)/0x100000000*uuidNanosPerMillisecond) & 0xfffffff
	out[4] = byte(high >> 8)            // #nosec G115 -- narrowed to the width the layout gives the field
	out[5] = byte(high)                 // #nosec G115 -- narrowed to the width the layout gives the field
	out[6] = byte(high>>24)&0x0f | 0x10 // #nosec G115 -- narrowed to the width the layout gives the field
	out[7] = byte(high >> 16)           // #nosec G115 -- narrowed to the width the layout gives the field

	out[8] = byte(clockseq>>8) | 0x80 // #nosec G115 -- narrowed to the width the layout gives the field
	out[9] = byte(clockseq)           // #nosec G115 -- narrowed to the width the layout gives the field
	copy(out[10:], node[:6])
	return out
}

// uuidV1ToV6 rearranges a version 1 UUID so its timestamp reads most
// significant digits first, which makes the written form sort by time.
func uuidV1ToV6(b []byte) []byte {
	return []byte{
		b[6]&0x0f<<4 | b[7]>>4&0x0f,
		b[7]&0x0f<<4 | b[4]&0xf0>>4,
		b[4]&0x0f<<4 | b[5]&0xf0>>4,
		b[5]&0x0f<<4 | b[0]&0xf0>>4,
		b[0]&0x0f<<4 | b[1]&0xf0>>4,
		b[1]&0x0f<<4 | b[2]&0xf0>>4,
		0x60 | b[2]&0x0f,
		b[3], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15],
	}
}

// uuidV7Bytes lays out a version 7 UUID: milliseconds since the Unix epoch, a
// sequence that keeps UUIDs made in the same millisecond in order, and
// randomness for the rest.
func uuidV7Bytes(random []byte, msecs, seq int64) []byte {
	out := make([]byte, uuidSize)
	out[0] = byte(msecs >> 40)               // #nosec G115 -- narrowed to the width the layout gives the field
	out[1] = byte(msecs >> 32)               // #nosec G115 -- narrowed to the width the layout gives the field
	out[2] = byte(msecs >> 24)               // #nosec G115 -- narrowed to the width the layout gives the field
	out[3] = byte(msecs >> 16)               // #nosec G115 -- narrowed to the width the layout gives the field
	out[4] = byte(msecs >> 8)                // #nosec G115 -- narrowed to the width the layout gives the field
	out[5] = byte(msecs)                     // #nosec G115 -- narrowed to the width the layout gives the field
	out[6] = 0x70 | byte(seq>>28)&0x0f       // #nosec G115 -- narrowed to the width the layout gives the field
	out[7] = byte(seq >> 20)                 // #nosec G115 -- narrowed to the width the layout gives the field
	out[8] = 0x80 | byte(seq>>14)&0x3f       // #nosec G115 -- narrowed to the width the layout gives the field
	out[9] = byte(seq >> 6)                  // #nosec G115 -- narrowed to the width the layout gives the field
	out[10] = byte(seq<<2) | random[10]&0x03 // #nosec G115 -- narrowed to the width the layout gives the field
	copy(out[11:], random[11:16])
	return out
}

// uuidStringify writes the bytes in the usual five groups.
func uuidStringify(b []byte) string {
	s := hex.EncodeToString(b)
	return s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]
}

// uuidClock carries what the timestamped versions have to remember between
// calls so that two UUIDs made in the same millisecond still differ and still
// sort in the order they were made.
type uuidClock struct {
	mu sync.Mutex

	v1Msecs    int64
	v1Nsecs    int
	v1Node     []byte
	v1Clockseq int

	v6Msecs int64
	v6Nsecs int

	v7Msecs int64
	v7Seq   int64
}

// newUUIDClock starts a clock that has not yet seen any moment, so the first
// UUID of each version is treated as beginning a fresh millisecond.
func newUUIDClock() *uuidClock {
	return &uuidClock{v1Msecs: math.MinInt64, v6Msecs: math.MinInt64, v7Msecs: math.MinInt64}
}

var uuidState = newUUIDClock()

// nextV1 advances the version 1 clock and lays out the next UUID. The node is
// settled once and kept, with the bit set that says it was chosen locally
// rather than read off a network card. It is given up when the sub-millisecond
// counter runs out or the clock goes backwards, either of which could otherwise
// let the same UUID come round twice.
func (c *uuidClock) nextV1(now int64, random []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch {
	case now == c.v1Msecs:
		c.v1Nsecs++
		if c.v1Nsecs >= uuidMaxNanos {
			c.v1Node = nil
			c.v1Nsecs = 0
		}
	case now > c.v1Msecs:
		c.v1Nsecs = 0
	default:
		c.v1Node = nil
	}
	if c.v1Node == nil {
		c.v1Node = append([]byte(nil), random[10:16]...)
		c.v1Node[0] |= 0x01
		c.v1Clockseq = (int(random[8])<<8 | int(random[9])) & 0x3fff
	}
	c.v1Msecs = now

	return uuidV1Bytes(random, now, c.v1Nsecs, c.v1Clockseq, c.v1Node)
}

// nextV6 advances the version 6 clock. Unlike version 1 it takes a fresh node
// and clock sequence every time, so nothing about the machine is carried from
// one UUID to the next.
func (c *uuidClock) nextV6(now int64, random []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if now == c.v6Msecs {
		c.v6Nsecs++
		if c.v6Nsecs >= uuidMaxNanos {
			c.v6Nsecs = 0
		}
	} else if now > c.v6Msecs {
		c.v6Nsecs = 0
	}
	c.v6Msecs = now

	return uuidV1Bytes(random, now, c.v6Nsecs, uuidDrawClockseq, nil)
}

// nextV7 advances the version 7 sequence. Within one millisecond it counts up,
// so the UUIDs sort in the order they were made; a new millisecond starts the
// sequence somewhere random. Should the sequence ever turn all the way over it
// borrows a millisecond from ahead, which keeps the order intact.
func (c *uuidClock) nextV7(now int64, random []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if now > c.v7Msecs {
		// #nosec G115 -- the sequence is a signed 32-bit counter by design, and
		// wraps rather than growing without bound.
		c.v7Seq = int64(int32(uint32(random[6])<<23 |
			uint32(random[7])<<16 | uint32(random[8])<<8 | uint32(random[9])))
		c.v7Msecs = now
	} else {
		c.v7Seq = int64(int32(c.v7Seq + 1)) // #nosec G115 -- as above
		if c.v7Seq == 0 {
			c.v7Msecs++
		}
	}

	return uuidV7Bytes(random, c.v7Msecs, c.v7Seq)
}

// uuidNow is the moment a UUID is being made, counted the way the timestamped
// versions count.
func uuidNow() int64 { return time.Now().UnixMilli() }

// uuidRandomBytes fills b with randomness. It is a package variable so tests can
// stand in for the system's source of it.
var uuidRandomBytes = func(b []byte) error {
	_, err := rand.Read(b)
	return err
}

// uuidRandom draws the bytes a UUID is built from.
func uuidRandom() ([]byte, error) {
	b := make([]byte, uuidSize)
	if err := uuidRandomBytes(b); err != nil {
		return nil, err
	}
	return b, nil
}

func init() { core.Register(GenerateUUID{}) }
