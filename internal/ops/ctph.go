package ops

import (
	"math"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(CTPH{})
	core.Register(CompareCTPHHashes{})
}

const (
	ctphHashPrime = 0x01000193 // FNV prime
	ctphHashInit  = 0x28021967 // FNV offset basis used by ctph.js
	ctphMinBlock  = 3          // minimum block-size multiplier
	ctphMinSig    = 32         // target minimum length of the first signature
	ctphRollWin   = 7          // rolling-hash window size
)

// ctphFNV is ctph.js's FNV-1 step: ((base * PRIME) ^ b) >>> 0, with the
// multiplication done in float64 exactly as the JavaScript does. That matters
// because the product exceeds 2^53 for large accumulators, losing low bits —
// a quirk jsToUint32 (ToUint32) preserves.
func ctphFNV(base uint32, b byte) uint32 {
	return jsToUint32(float64(base)*ctphHashPrime) ^ uint32(b)
}

// ctphRollHash is ctph.js's rolling hash over a 7-byte window.
type ctphRollHash struct {
	x, y   int64
	z      uint32
	c      int
	window [ctphRollWin]int64
}

func (r *ctphRollHash) update(d byte) {
	r.y -= r.x
	r.y += 7 * int64(d)
	r.x += int64(d)
	r.x -= r.window[r.c%ctphRollWin]
	r.window[r.c%ctphRollWin] = int64(d)
	r.c++
	r.z <<= 5
	r.z ^= uint32(d)
}

func (r *ctphRollHash) sum() uint32 {
	return jsToUint32(float64(r.x) + float64(r.y) + float64(r.z))
}

// ctphPiecewise produces the two piecewise signatures for a trigger value.
func ctphPiecewise(bytes []byte, trigger uint32) [2]string {
	var sig [2]string
	h1, h2 := uint32(ctphHashInit), uint32(ctphHashInit)
	var rh ctphRollHash
	for i, b := range bytes {
		h1 = ctphFNV(h1, b)
		h2 = ctphFNV(h2, b)
		rh.update(b)
		last := i == len(bytes)-1
		if last || rh.sum()%trigger == trigger-1 {
			sig[0] += string(fuzzyB64[h1&63])
			h1 = ctphHashInit
		}
		if last || rh.sum()%(trigger*2) == trigger*2-1 {
			sig[1] += string(fuzzyB64[h2&63])
			h2 = ctphHashInit
		}
	}
	return sig
}

// ctphDigest computes the CTPH fuzzy hash of the input bytes.
func ctphDigest(bytes []byte) string {
	bi := int(math.Ceil(math.Log(float64(len(bytes))/float64(64*ctphMinBlock)) / math.Log(2)))
	bi = max(ctphMinBlock, bi)
	sig := ctphPiecewise(bytes, uint32(ctphMinBlock<<bi)) // #nosec G115 -- block index is small
	for bi > 0 && len(sig[0]) < ctphMinSig {
		bi--
		sig = ctphPiecewise(bytes, uint32(ctphMinBlock<<bi)) // #nosec G115 -- block index is small
	}
	return string(fuzzyB64[bi]) + ":" + sig[0] + ":" + sig[1]
}

// CTPH computes a Context Triggered Piecewise Hash (fuzzy hash). Ported from
// CyberChef CTPH.mjs (which wraps the non-standard ctph.js package); the
// algorithm is reimplemented from that package to match its output.
type CTPH struct{}

// Meta returns the operation metadata.
func (CTPH) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CTPH",
		Module:      "Crypto",
		Description: "Context Triggered Piecewise Hashing, also called Fuzzy Hashing, can match inputs that have homologies. Such inputs have sequences of identical bytes in the same order, although bytes in between these sequences may be different in both content and length.<br><br>CTPH was originally based on the work of Dr. Andrew Tridgell and a spam email detector called SpamSum. This method was adapted by Jesse Kornblum and published at the DFRWS conference in 2006 in a paper 'Identifying Almost Identical Files Using Context Triggered Piecewise Hashing'.",
		InfoURL:     "https://forensics.wiki/context_triggered_piecewise_hashing/",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CTPH) Args() []core.ArgDef { return nil }

// Run computes the CTPH digest.
func (CTPH) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish([]byte(ctphDigest(in.Bytes())), core.TypeString), nil
}

// CompareCTPHHashes compares two CTPH hashes, returning a 0–100 similarity.
// Ported from CyberChef CompareCTPHHashes.mjs.
type CompareCTPHHashes struct{}

// Meta returns the operation metadata.
func (CompareCTPHHashes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Compare CTPH hashes",
		Module:      "Crypto",
		Description: "Compares two Context Triggered Piecewise Hashing (CTPH) fuzzy hashes to determine the similarity between them on a scale of 0 to 100.",
		InfoURL:     "https://forensics.wiki/context_triggered_piecewise_hashing/",
		InputType:   core.TypeString,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (CompareCTPHHashes) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: []string{"Line feed", "CRLF", "Space", "Comma"}},
	}
}

// Run compares the two hashes.
func (CompareCTPHHashes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := fuzzyCompare(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeNumber), nil
}
