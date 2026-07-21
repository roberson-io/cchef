package ops

import (
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(SSDEEP{})
	core.Register(CompareSSDEEPHashes{})
}

const (
	ssdeepHashPrime = 16777619  // FNV prime
	ssdeepHashInit  = 671226215 // FNV offset basis used by ssdeep.js
	ssdeepRollWin   = 7         // rolling-hash window size
	ssdeepMaxLen    = 64        // maximum length of the first signature
)

// jsInt32 reinterprets an ECMAScript number as a signed 32-bit integer (ToInt32).
func jsInt32(f float64) int32 { return int32(jsToUint32(f)) } // #nosec G115 -- ToInt32 reinterprets the 32-bit pattern

// ssdeepSafeAdd is ssdeep.js's safe_add: 32-bit addition performed with 16-bit
// halves, exactly as the JavaScript does (all bitwise ops are ToInt32).
func ssdeepSafeAdd(xf, yf float64) uint32 {
	x, y := int64(jsInt32(xf)), int64(jsInt32(yf))
	lsw := (x & 0xFFFF) + (y & 0xFFFF)
	msw := (x >> 16) + (y >> 16) + (lsw >> 16)
	return uint32((msw << 16) | (lsw & 0xFFFF)) // #nosec G115 -- 32-bit result, matching JS bitwise semantics
}

// ssdeepSafeMultiply is ssdeep.js's safe_multiply: an exact 32-bit multiply built
// from 16-bit partial products (avoids the float-precision loss CTPH has).
func ssdeepSafeMultiply(xf, yf float64) uint32 {
	x, y := int64(jsInt32(xf)), int64(jsInt32(yf))
	xlsw := x & 0xFFFF
	xmsw := (x >> 16) + (xlsw >> 16)
	ylsw := y & 0xFFFF
	ymsw := (y >> 16) + (ylsw >> 16)
	c00 := xlsw * ylsw
	c16 := c00 >> 16
	c16 += xmsw * ylsw
	c16 &= 0xFFFF
	c16 += xlsw * ymsw
	out := ((c16 & 0xFFFF) << 16) | (c00 & 0xFFFF)
	return uint32(int32(out)) // #nosec G115 -- 32-bit result, matching JS bitwise semantics
}

// ssdeepFNV is ssdeep.js's FNV-1 step.
func ssdeepFNV(h uint32, c byte) uint32 {
	return ssdeepSafeMultiply(float64(h), ssdeepHashPrime) ^ uint32(c)
}

// ssdeepRollHash is ssdeep.js's rolling hash over a 7-byte window.
type ssdeepRollHash struct {
	h1, h2, h3 uint32
	n          int
	window     [ssdeepRollWin]uint32
}

func (r *ssdeepRollHash) update(c byte) {
	r.h2 = ssdeepSafeAdd(float64(r.h2), -float64(r.h1))
	r.h2 = ssdeepSafeAdd(float64(r.h2), float64(ssdeepRollWin*int(c)))
	r.h1 = ssdeepSafeAdd(float64(r.h1), float64(c))
	val := r.window[r.n%ssdeepRollWin]
	r.h1 = ssdeepSafeAdd(float64(r.h1), -float64(val))
	r.window[r.n%ssdeepRollWin] = uint32(c)
	r.n++
	r.h3 = (r.h3 << 5) ^ uint32(c)
}

func (r *ssdeepRollHash) sum() uint32 {
	return jsToUint32(float64(r.h1) + float64(r.h2) + float64(r.h3))
}

// ssdeepPiecewise produces the two signatures and the trigger for one block size.
func ssdeepPiecewise(bytes []byte, trigger int) (sig0, sig1 string) {
	if len(bytes) == 0 {
		return "", ""
	}
	h1, h2 := uint32(ssdeepHashInit), uint32(ssdeepHashInit)
	var rh ssdeepRollHash
	t := uint32(trigger) // #nosec G115 -- trigger is a small positive block size
	for _, b := range bytes {
		h1 = ssdeepFNV(h1, b)
		h2 = ssdeepFNV(h2, b)
		rh.update(b)
		if len(sig0) < ssdeepMaxLen-1 && rh.sum()%t == t-1 {
			sig0 += string(fuzzyB64[h1&63])
			h1 = ssdeepHashInit
		}
		if len(sig1) < ssdeepMaxLen/2-1 && rh.sum()%(t*2) == t*2-1 {
			sig1 += string(fuzzyB64[h2&63])
			h2 = ssdeepHashInit
		}
	}
	sig0 += string(fuzzyB64[h1&63])
	sig1 += string(fuzzyB64[h2&63])
	return sig0, sig1
}

// ssdeepDigest computes the SSDEEP fuzzy hash of the input bytes.
func ssdeepDigest(bytes []byte) string {
	bi := 3
	for bi*ssdeepMaxLen < len(bytes) {
		bi *= 2
	}
	var sig0, sig1 string
	var trigger int
	for {
		trigger = bi
		sig0, sig1 = ssdeepPiecewise(bytes, bi)
		bi /= 2
		if bi <= 3 || len(sig0) >= ssdeepMaxLen/2 {
			break
		}
	}
	return strconv.Itoa(trigger) + ":" + sig0 + ":" + sig1
}

// SSDEEP computes an SSDEEP fuzzy hash. Ported from CyberChef SSDEEP.mjs (which
// wraps the non-standard ssdeep.js package); the algorithm is reimplemented from
// that package to match its output.
type SSDEEP struct{}

// Meta returns the operation metadata.
func (SSDEEP) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SSDEEP",
		Module:      "Crypto",
		Description: "SSDEEP is a program for computing context triggered piecewise hashes (CTPH). Also called fuzzy hashes, CTPH can match inputs that have homologies. Such inputs have sequences of identical bytes in the same order, although bytes in between these sequences may be different in both content and length.<br><br>SSDEEP hashes are now widely used for simple identification purposes (e.g. the 'Basic Properties' section in VirusTotal). Although 'better' fuzzy hashes are available, SSDEEP is still one of the primary choices because of its speed and being a de facto standard.<br><br>This operation is fundamentally the same as the CTPH operation, however their outputs differ in format.",
		InfoURL:     "https://forensics.wiki/ssdeep",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SSDEEP) Args() []core.ArgDef { return nil }

// Run computes the SSDEEP digest.
func (SSDEEP) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return core.NewDish([]byte(ssdeepDigest(in.Bytes())), core.TypeString), nil
}

// CompareSSDEEPHashes compares two SSDEEP hashes, returning a 0–100 similarity.
// Ported from CyberChef CompareSSDEEPHashes.mjs.
type CompareSSDEEPHashes struct{}

// Meta returns the operation metadata.
func (CompareSSDEEPHashes) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Compare SSDEEP hashes",
		Module:      "Crypto",
		Description: "Compares two SSDEEP fuzzy hashes to determine the similarity between them on a scale of 0 to 100.",
		InfoURL:     "https://forensics.wiki/ssdeep/",
		InputType:   core.TypeString,
		OutputType:  core.TypeNumber,
	}
}

// Args returns the argument definitions.
func (CompareSSDEEPHashes) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Delimiter", Type: core.ArgOption, Value: []string{"Line feed", "CRLF", "Space", "Comma"}},
	}
}

// Run compares the two hashes.
func (CompareSSDEEPHashes) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := fuzzyCompare(in.String(), args[0].(string))
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeNumber), nil
}
