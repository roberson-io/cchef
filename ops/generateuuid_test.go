package ops

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/roberson-io/cchef/core"
)

// defaultUUIDNamespace is the namespace the operation offers to begin with.
const defaultUUIDNamespace = "1b671a64-40d5-491e-99b0-da01ff1f3341"

// makeUUID makes one UUID of the version asked for.
func makeUUID(t *testing.T, version, namespace, name string) string {
	t.Helper()
	out, err := core.Recipe{{Op: "Generate UUID", Args: []any{version, namespace}}}.
		Execute(core.NewDish([]byte(name), core.TypeString))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out.String()
}

// analyseUUID reads a UUID back, so a generated one can be checked against what
// it claims to hold.
func analyseUUID(t *testing.T, id string) string {
	t.Helper()
	out, err := analyseUUIDRecipe(true).Execute(core.NewDish([]byte(id), core.TypeString))
	if err != nil {
		t.Fatalf("Analyse UUID(%s): %v", id, err)
	}
	return out.String()
}

// uuidField pulls one labelled value out of an analysis.
func uuidField(t *testing.T, analysis, label string) string {
	t.Helper()
	for section := range strings.SplitSeq(analysis, "\n\n") {
		if name, value, ok := strings.Cut(section, ":\n"); ok && name == label {
			return value
		}
	}
	t.Fatalf("no %q in:\n%s", label, analysis)
	return ""
}

// uuidNameVector is one recorded name-based UUID: the version, the namespace it
// was derived under, the name in hexadecimal so any bytes may appear, and what
// the uuid package makes of them.
type uuidNameVector struct {
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Want      string `json:"want"`
}

// TestGenerateUUIDNameBased covers versions 3 and 5, which hash a name under a
// namespace and so give the same answer every time. The vectors come from the
// uuid package CyberChef calls.
func TestGenerateUUIDNameBased(t *testing.T) {
	file, err := os.Open("testdata/generate_uuid.jsonl")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer func() { _ = file.Close() }()

	seen := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var v uuidNameVector
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatalf("parse vector: %v", err)
		}
		name, err := hex.DecodeString(v.Name)
		if err != nil {
			t.Fatalf("decode name: %v", err)
		}
		seen++
		t.Run(v.Version+"/"+v.Namespace+"/"+v.Name, func(t *testing.T) {
			if got := makeUUID(t, v.Version, v.Namespace, string(name)); got != v.Want {
				t.Errorf("got %s, want %s", got, v.Want)
			}
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	if seen == 0 {
		t.Fatal("no vectors were read")
	}
}

// uuidByteVector is one recorded call to a byte builder with every random or
// time-derived input supplied, so the answer is fixed. Recorded from the uuid
// package, which accepts exactly these as options.
type uuidByteVector struct {
	Kind     string `json:"kind"`
	Random   string `json:"random"`
	Msecs    int64  `json:"msecs"`
	Nsecs    int    `json:"nsecs"`
	Seq      int64  `json:"seq"`
	Clockseq int    `json:"clockseq"`
	Node     string `json:"node"`
	Want     string `json:"want"`
}

// TestUUIDByteBuilders covers the layouts the timestamped versions pack their
// bytes into, which is where the awkward arithmetic lives: a version 1
// timestamp is split across three fields at three different widths, and a
// version 7 sequence is spread over five bytes.
func TestUUIDByteBuilders(t *testing.T) {
	file, err := os.Open("testdata/uuid_bytes.jsonl")
	if err != nil {
		t.Fatalf("open vectors: %v", err)
	}
	defer func() { _ = file.Close() }()

	counts := map[string]int{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var v uuidByteVector
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			t.Fatalf("parse vector: %v", err)
		}
		random := mustHex(t, v.Random)
		counts[v.Kind]++

		var got string
		switch v.Kind {
		case "v1":
			got = uuidStringify(uuidV1Bytes(random, v.Msecs, v.Nsecs, v.Clockseq, mustHex(t, v.Node)))
		case "v6":
			got = uuidStringify(uuidV1ToV6(uuidV1Bytes(random, v.Msecs, v.Nsecs, v.Clockseq, mustHex(t, v.Node))))
		case "v7":
			got = uuidStringify(uuidV7Bytes(random, v.Msecs, v.Seq))
		case "v4":
			got = uuidStringify(uuidV4Bytes(random))
		default:
			t.Fatalf("unknown vector kind %q", v.Kind)
		}
		if got != v.Want {
			t.Errorf("%s(%s, msecs=%d, nsecs=%d, seq=%d, clockseq=%d, node=%s):\n got %s\nwant %s",
				v.Kind, v.Random, v.Msecs, v.Nsecs, v.Seq, v.Clockseq, v.Node, got, v.Want)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	for _, kind := range []string{"v1", "v4", "v6", "v7"} {
		if counts[kind] == 0 {
			t.Errorf("no %s vectors were read", kind)
		}
	}
}

// TestGenerateUUIDLayout covers what every version has in common: the written
// shape, the version digit, and the variant the standard reserves.
func TestGenerateUUIDLayout(t *testing.T) {
	layout := regexp.MustCompile(
		`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for version, digit := range map[string]string{
		"v1": "1", "v3": "3", "v4": "4", "v5": "5", "v6": "6", "v7": "7",
	} {
		t.Run(version, func(t *testing.T) {
			id := makeUUID(t, version, defaultUUIDNamespace, "name")
			if !layout.MatchString(id) {
				t.Fatalf("%s is not laid out as a UUID", id)
			}
			if got := uuidField(t, analyseUUID(t, id), "Version"); got != digit {
				t.Errorf("version reads as %s, want %s", got, digit)
			}
		})
	}
}

// TestGenerateUUIDRandomVersionsDiffer covers the versions that draw on
// randomness, which must not repeat themselves.
func TestGenerateUUIDRandomVersionsDiffer(t *testing.T) {
	for _, version := range []string{"v1", "v4", "v6", "v7"} {
		t.Run(version, func(t *testing.T) {
			seen := map[string]bool{}
			for range 50 {
				id := makeUUID(t, version, defaultUUIDNamespace, "")
				if seen[id] {
					t.Fatalf("%s came up twice", id)
				}
				seen[id] = true
			}
		})
	}
}

// TestGenerateUUIDTimestamps covers the three versions that record when they
// were made. Each should read back as the moment it was generated.
func TestGenerateUUIDTimestamps(t *testing.T) {
	// Enough room for a slow machine to run the two steps, and far short of
	// anything that would hide a wrong epoch or a wrong unit.
	const tolerance = 5 * time.Second

	for _, version := range []string{"v1", "v6", "v7"} {
		t.Run(version, func(t *testing.T) {
			before := time.Now().UnixMilli()
			id := makeUUID(t, version, defaultUUIDNamespace, "")
			after := time.Now().UnixMilli()

			recorded, err := strconv.ParseInt(uuidField(t, analyseUUID(t, id), "Timestamp"), 10, 64)
			if err != nil {
				t.Fatalf("timestamp: %v", err)
			}
			if recorded < before-tolerance.Milliseconds() || recorded > after+tolerance.Milliseconds() {
				t.Errorf("recorded %d, which is outside %d..%d", recorded, before, after)
			}
		})
	}
}

// TestGenerateUUIDV1NodeIsSteady covers the node a version 1 UUID carries: the
// uuid package settles on one for the life of the process and marks it as not
// belonging to any real network card, where version 6 draws a fresh one each
// time.
func TestGenerateUUIDV1NodeIsSteady(t *testing.T) {
	nodes := map[string]int{}
	for range 10 {
		node := uuidField(t, analyseUUID(t, makeUUID(t, "v1", defaultUUIDNamespace, "")), "Node")
		nodes[node]++
	}
	if len(nodes) != 1 {
		t.Errorf("version 1 used %d different nodes, want 1: %v", len(nodes), nodes)
	}
	for node := range nodes {
		first, err := strconv.ParseUint(node[:2], 16, 8)
		if err != nil {
			t.Fatalf("node %q: %v", node, err)
		}
		if first&0x01 == 0 {
			t.Errorf("node %s is not marked as locally chosen", node)
		}
	}

	fresh := map[string]bool{}
	for range 10 {
		fresh[uuidField(t, analyseUUID(t, makeUUID(t, "v6", defaultUUIDNamespace, "")), "Node")] = true
	}
	if len(fresh) < 2 {
		t.Errorf("version 6 reused its node across 10 UUIDs")
	}
}

// TestGenerateUUIDV7SortsByTime covers the ordering version 7 promises: UUIDs
// made one after another sort into the order they were made, even when several
// land in the same millisecond.
func TestGenerateUUIDV7SortsByTime(t *testing.T) {
	ids := make([]string, 20)
	for i := range ids {
		ids[i] = makeUUID(t, "v7", defaultUUIDNamespace, "")
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("%s does not sort after %s", ids[i], ids[i-1])
		}
	}
}

// TestGenerateUUIDNamespaceRejected covers the namespace check, which only the
// two name-based versions apply.
func TestGenerateUUIDNamespaceRejected(t *testing.T) {
	for _, version := range []string{"v3", "v5"} {
		for _, namespace := range []string{"", "not-a-uuid", "1b671a64-40d5-491e-99b0-da01ff1f334"} {
			t.Run(version+"/"+namespace, func(t *testing.T) {
				out, err := runOp(t, "Generate UUID", "name", version, namespace)
				if err == nil {
					t.Fatalf("accepted namespace %q, giving %s", namespace, out)
				}
				if err.Error() != "Invalid UUID namespace" {
					t.Errorf("got %q, want %q", err.Error(), "Invalid UUID namespace")
				}
			})
		}
	}
}

// TestGenerateUUIDNamespaceIgnored covers the versions that take no namespace,
// which should not mind one that could never be read.
func TestGenerateUUIDNamespaceIgnored(t *testing.T) {
	for _, version := range []string{"v1", "v4", "v6", "v7"} {
		t.Run(version, func(t *testing.T) {
			if _, err := runOp(t, "Generate UUID", "", version, "not-a-uuid"); err != nil {
				t.Errorf("version %s minded the namespace: %v", version, err)
			}
		})
	}
}

// v1Parts pulls the clock sequence and node out of a version 1 UUID, which is
// what the clock has to keep steady between calls.
func v1Parts(b []byte) (clockseq int, node string) {
	return int(b[8]&0x3f)<<8 | int(b[9]), hex.EncodeToString(b[10:16])
}

// uuidTestRandom is a fixed draw, so a clock driven by hand gives a fixed
// answer. The bytes differ from one another so a wrong slice shows up.
func uuidTestRandom(seed byte) []byte {
	b := make([]byte, uuidSize)
	for i := range b {
		b[i] = seed + byte(i)*7
	}
	return b
}

// TestUUIDClockV1 covers what a version 1 clock has to remember. Two UUIDs made
// in the same millisecond are told apart by a counter, and the node is settled
// once and then kept — except when the counter runs out or the clock goes
// backwards, either of which starts again with a fresh node.
func TestUUIDClockV1(t *testing.T) {
	t.Run("the node and clock sequence stay put", func(t *testing.T) {
		c := newUUIDClock()
		_, first := v1Parts(c.nextV1(1000, uuidTestRandom(1)))
		clockseq, node := v1Parts(c.nextV1(1000, uuidTestRandom(9)))
		if node != first {
			t.Errorf("node changed from %s to %s", first, node)
		}
		if _, again := v1Parts(c.nextV1(1001, uuidTestRandom(9))); again != first {
			t.Errorf("node changed into the next millisecond: %s", again)
		}
		if want := (int(uuidTestRandom(1)[8])<<8 | int(uuidTestRandom(1)[9])) & 0x3fff; clockseq != want {
			t.Errorf("clock sequence %d, want %d", clockseq, want)
		}
	})

	t.Run("the counter runs out and the node starts again", func(t *testing.T) {
		c := newUUIDClock()
		_, first := v1Parts(c.nextV1(1000, uuidTestRandom(1)))
		var node string
		for i := range uuidMaxNanos {
			_, node = v1Parts(c.nextV1(1000, uuidTestRandom(byte(i))))
		}
		if node == first {
			t.Error("the node survived the counter running out")
		}
	})

	t.Run("the clock goes backwards and the node starts again", func(t *testing.T) {
		c := newUUIDClock()
		_, first := v1Parts(c.nextV1(1000, uuidTestRandom(1)))
		if _, node := v1Parts(c.nextV1(999, uuidTestRandom(2))); node == first {
			t.Error("the node survived the clock going backwards")
		}
	})

	t.Run("the counter climbs within one millisecond", func(t *testing.T) {
		c := newUUIDClock()
		seen := map[string]bool{}
		for range 5 {
			seen[hex.EncodeToString(c.nextV1(1000, uuidTestRandom(1)))] = true
		}
		if len(seen) != 5 {
			t.Errorf("got %d distinct UUIDs from 5 calls in one millisecond, want 5", len(seen))
		}
	})
}

// TestUUIDClockV6 covers the version 6 clock, which counts within a millisecond
// the same way but draws a fresh node every time rather than keeping one.
func TestUUIDClockV6(t *testing.T) {
	c := newUUIDClock()
	seen := map[string]bool{}
	for range 5 {
		seen[hex.EncodeToString(c.nextV6(1000, uuidTestRandom(1)))] = true
	}
	if len(seen) != 5 {
		t.Errorf("got %d distinct UUIDs from 5 calls in one millisecond, want 5", len(seen))
	}

	// The counter wraps rather than reaching for a new node.
	c = newUUIDClock()
	for range uuidMaxNanos + 1 {
		c.nextV6(1000, uuidTestRandom(1))
	}
	if c.v6Nsecs != 0 {
		t.Errorf("counter reads %d after running out, want 0", c.v6Nsecs)
	}
}

// TestUUIDClockV7 covers the version 7 sequence: a fresh millisecond draws a
// starting point at random, and within one millisecond the sequence counts up so
// the UUIDs sort in the order they were made.
func TestUUIDClockV7(t *testing.T) {
	t.Run("the sequence counts up within a millisecond", func(t *testing.T) {
		c := newUUIDClock()
		var previous string
		for i := range 5 {
			id := hex.EncodeToString(c.nextV7(1000, uuidTestRandom(1)))
			if i > 0 && id <= previous {
				t.Errorf("%s does not sort after %s", id, previous)
			}
			previous = id
		}
	})

	t.Run("a new millisecond draws a fresh sequence", func(t *testing.T) {
		c := newUUIDClock()
		c.nextV7(1000, uuidTestRandom(1))
		c.nextV7(1001, uuidTestRandom(2))
		random := uuidTestRandom(2)
		want := int64(int32(uint32(random[6])<<23 | uint32(random[7])<<16 |
			uint32(random[8])<<8 | uint32(random[9])))
		if c.v7Seq != want {
			t.Errorf("sequence %d, want %d", c.v7Seq, want)
		}
	})

	t.Run("the sequence turns over and takes the millisecond with it", func(t *testing.T) {
		// The sequence starts somewhere in the lower half of the range and
		// counts up one at a time, so reaching this by hand would take billions
		// of calls inside a single millisecond. It is driven straight instead.
		c := newUUIDClock()
		c.v7Msecs, c.v7Seq = 1000, -1
		c.nextV7(1000, uuidTestRandom(1))
		if c.v7Seq != 0 || c.v7Msecs != 1001 {
			t.Errorf("sequence %d at millisecond %d, want 0 at 1001", c.v7Seq, c.v7Msecs)
		}
	})
}

// TestGenerateUUIDNoRandomness covers what happens when the system will not give
// out random bytes, which the versions that need them cannot work around.
func TestGenerateUUIDNoRandomness(t *testing.T) {
	original := uuidRandomBytes
	uuidRandomBytes = func([]byte) error { return errors.New("no entropy") }
	defer func() { uuidRandomBytes = original }()

	for _, version := range []string{"v1", "v4", "v6", "v7"} {
		t.Run(version, func(t *testing.T) {
			if _, err := runOp(t, "Generate UUID", "", version, defaultUUIDNamespace); err == nil {
				t.Error("made a UUID with no randomness to draw on")
			}
		})
	}
	t.Run("v5, which needs none", func(t *testing.T) {
		if _, err := runOp(t, "Generate UUID", "", "v5", defaultUUIDNamespace); err != nil {
			t.Errorf("a name-based UUID wanted randomness: %v", err)
		}
	})
}

// TestGenerateUUIDUnknownVersion covers the guard on the version. The recipe
// engine checks the option before the operation runs, so the guard only answers
// a direct call.
func TestGenerateUUIDUnknownVersion(t *testing.T) {
	if _, err := (GenerateUUID{}).Run(
		core.NewDish([]byte("name"), core.TypeString),
		[]any{"v2", defaultUUIDNamespace},
	); err == nil {
		t.Error("accepted a version the operation does not offer")
	}
}
