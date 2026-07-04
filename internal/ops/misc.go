package ops

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseObjectIDTimestamp{})
	core.Register(Sleep{})
	core.Register(FileTree{})
	core.Register(Shuffle{})
	core.Register(PRNG{})
}

// ParseObjectIDTimestamp extracts the embedded timestamp from a MongoDB ObjectID.
type ParseObjectIDTimestamp struct{}

// Meta returns the operation metadata.
func (ParseObjectIDTimestamp) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse ObjectID timestamp",
		Module:      "Default",
		Description: "Extracts the timestamp from a MongoDB ObjectID (the first 4 bytes are a Unix timestamp).",
		InfoURL:     "https://docs.mongodb.com/manual/reference/method/ObjectId.getTimestamp/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseObjectIDTimestamp) Args() []core.ArgDef { return nil }

// Run parses the timestamp. Ported from CyberChef ParseObjectIDTimestamp.mjs.
func (ParseObjectIDTimestamp) Run(in *core.Dish, args []any) (*core.Dish, error) {
	s := strings.TrimSpace(in.String())
	if len(s) != 24 {
		return nil, fmt.Errorf("input must be a 24-character hex ObjectID")
	}
	secs, err := strconv.ParseInt(s[:8], 16, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ObjectID: %w", err)
	}
	iso := time.Unix(secs, 0).UTC().Format("2006-01-02T15:04:05.000Z")
	return core.NewDish([]byte(iso), core.TypeString), nil
}

// Sleep delays for the given number of milliseconds, then passes the input on.
type Sleep struct{}

// Meta returns the operation metadata.
func (Sleep) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Sleep",
		Module:      "Default",
		Description: "Pauses execution for the given number of milliseconds, then passes the input through unchanged.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (Sleep) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Time (ms)", Type: core.ArgNumber, Value: 1000},
	}
}

// Run sleeps then returns the input. Ported from CyberChef Sleep.mjs.
func (Sleep) Run(in *core.Dish, args []any) (*core.Dish, error) {
	ms := int(args[0].(float64))
	if ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	return core.NewDish(in.Bytes(), core.TypeArrayBuffer), nil
}

// FileTree renders a list of file paths as a tree.
type FileTree struct{}

// Meta returns the operation metadata.
func (FileTree) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "File Tree",
		Module:      "Default",
		Description: "Creates a file tree from a list of file paths.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (FileTree) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "File Path Delimiter", Type: core.ArgString, Value: "/"},
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
	}
}

// Run builds the tree. Ported from CyberChef FileTree.mjs.
func (FileTree) Run(in *core.Dish, args []any) (*core.Dish, error) {
	const arrow, pipe = "|---", "|   "
	fileDelim := parseEscapedChars(args[0].(string))
	entryDelim := charRep(args[1].(string))

	paths := uniqueSorted(splitByDelim(in.String(), entryDelim))
	completed := map[string]bool{}
	var printList []string
	for _, fp := range paths {
		path := strings.Split(fp, fileDelim)
		if len(path) > 0 && path[0] == "" {
			path = path[1:]
		}
		for j, seg := range path {
			var printLine, key string
			if j == 0 {
				printLine, key = seg, seg
			} else {
				printLine = strings.Repeat(pipe, j-1) + arrow + seg
				key = strings.Join(path[:j+1], "/")
			}
			if !completed[key] {
				completed[key] = true
				printList = append(printList, printLine)
			}
		}
	}
	return core.NewDish([]byte(strings.Join(printList, "\n")), core.TypeString), nil
}

// uniqueSorted returns the sorted, de-duplicated elements of s.
func uniqueSorted(s []string) []string {
	cp := append([]string(nil), s...)
	sort.Strings(cp)
	out := cp[:0]
	for i, v := range cp {
		if i == 0 || v != cp[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// Shuffle randomly reorders the sections of the input.
type Shuffle struct{}

// Meta returns the operation metadata.
func (Shuffle) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Shuffle",
		Module:      "Default",
		Description: "Randomly reorders the sections of the input, split on the given delimiter.",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Shuffle) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: inputDelims},
	}
}

// Run shuffles the input. Ported from CyberChef Shuffle.mjs (Fisher-Yates with a
// cryptographic RNG).
func (Shuffle) Run(in *core.Dish, args []any) (*core.Dish, error) {
	if in.String() == "" {
		return core.NewDish(nil, core.TypeString), nil
	}
	delim := charRep(args[0].(string))
	parts := splitByDelim(in.String(), delim)
	for i := len(parts) - 1; i > 0; i-- {
		j := randInt(i + 1)
		parts[i], parts[j] = parts[j], parts[i]
	}
	return core.NewDish([]byte(strings.Join(parts, delim)), core.TypeString), nil
}

// randInt returns a uniform random integer in [0, bound) using crypto/rand.
func randInt(bound int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(bound)))
	if err != nil {
		panic(err) // crypto/rand failure is unrecoverable
	}
	return int(n.Int64())
}

// PRNG generates cryptographically-random data in the chosen format.
type PRNG struct{}

// Meta returns the operation metadata.
func (PRNG) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Pseudo-Random Number Generator",
		Module:      "Default",
		Description: "Generates a random number or sequence of bytes using a cryptographically-secure source.",
		InfoURL:     "https://wikipedia.org/wiki/Cryptographically_secure_pseudorandom_number_generator",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (PRNG) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Number of bytes", Type: core.ArgNumber, Value: 32},
		{Name: "Output as", Type: core.ArgOption, Value: []string{"Hex", "Integer", "Byte array", "Raw"}},
	}
}

// Run generates the random data. Ported from CyberChef PseudoRandomNumberGenerator.mjs.
func (PRNG) Run(in *core.Dish, args []any) (*core.Dish, error) {
	n := int(args[0].(float64))
	if n < 0 {
		return nil, fmt.Errorf("number of bytes must be non-negative")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	switch args[1].(string) {
	case "Hex":
		return core.NewDish([]byte(hex.EncodeToString(b)), core.TypeString), nil
	case "Integer":
		// Little-endian interpretation, matching CyberChef.
		v := new(big.Int)
		for i := len(b) - 1; i >= 0; i-- {
			v.Mul(v, big.NewInt(256))
			v.Add(v, big.NewInt(int64(b[i])))
		}
		return core.NewDish([]byte(v.Text(10)), core.TypeString), nil
	case "Byte array":
		nums := make([]string, len(b))
		for i, x := range b {
			nums[i] = strconv.Itoa(int(x))
		}
		return core.NewDish([]byte("["+strings.Join(nums, ",")+"]"), core.TypeString), nil
	default: // Raw
		return core.NewDish(b, core.TypeString), nil
	}
}
