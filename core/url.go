package core

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

// DefaultBaseURL is the public CyberChef instance share links point at unless
// the caller names another.
const DefaultBaseURL = "https://gchq.github.io/CyberChef/"

// fragmentSafe is the set of characters left unescaped in a URL fragment,
// matching CyberChef's Utils.encodeURIFragment (encodeURIComponent minus the
// re-allowed legal chars; &, + and = stay percent-encoded).
const fragmentSafe = "-._~!$'()*,;:@/?" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// EncodeURIFragment percent-encodes a string for use in a URL fragment, keeping
// the human-readable safe set literal. Ported from Utils.encodeURIFragment.
func EncodeURIFragment(s string) string {
	var sb strings.Builder
	for _, b := range []byte(s) {
		if strings.IndexByte(fragmentSafe, b) >= 0 {
			sb.WriteByte(b)
		} else {
			fmt.Fprintf(&sb, "%%%02X", b)
		}
	}
	return sb.String()
}

// DecodeURIFragment resolves the percent-escapes EncodeURIFragment writes,
// returning the bytes as they were. It is that function's inverse.
//
// A "+" is left alone rather than read as a space. CyberChef substitutes it,
// but EncodeURIFragment writes a space as %20 and a plus as %2B, so a literal
// "+" in a fragment can only have come from the data — most often from base64,
// where reading it as a space would lose a byte.
func DecodeURIFragment(s string) (string, error) {
	// PathUnescape rather than QueryUnescape, which is the one that would turn
	// "+" into a space.
	return url.PathUnescape(s)
}

// ParseURL reads a CyberChef share URL, returning the recipe it names and any
// input it carries. It accepts a whole URL, a bare "#..." fragment, or a bare
// parameter string. Settings only a browser can act on (theme, ienc, oenc,
// ieol, oeol) are ignored.
func ParseURL(u string) (Recipe, []byte, error) {
	params, err := parseURIParams(uriFragment(u))
	if err != nil {
		return nil, nil, err
	}
	if params["recipe"] == "" {
		return nil, nil, fmt.Errorf("no recipe in URL")
	}
	r, err := ParseRecipeConfig(params["recipe"])
	if err != nil {
		return nil, nil, err
	}
	// A recipe that names no operation cannot be run or shared onward, and the
	// recipe parser reports one for some malformed text rather than failing.
	if len(r) == 0 {
		return nil, nil, fmt.Errorf("no operations in the URL's recipe")
	}
	if params["input"] == "" {
		return r, nil, nil
	}
	in, err := decodeShareInput(params["input"])
	if err != nil {
		return nil, nil, err
	}
	return r, in, nil
}

// uriFragment returns a share URL's parameter section: everything after the
// first "#", or the query string when there is no fragment.
func uriFragment(u string) string {
	if _, after, ok := strings.Cut(u, "#"); ok {
		return after
	}
	if _, after, ok := strings.Cut(u, "?"); ok {
		return after
	}
	return u
}

// parseURIParams splits a parameter section into its "&"-separated pairs,
// percent-decoding each value. A pair's value may itself contain "=", which
// CyberChef's own reader mishandles; splitting only on the first one keeps a
// value that carries base64 padding readable.
func parseURIParams(params string) (map[string]string, error) {
	out := make(map[string]string)
	if params == "" {
		return out, nil
	}
	for p := range strings.SplitSeq(params, "&") {
		key, value, found := strings.Cut(p, "=")
		if !found {
			continue // a bare flag carries no value to read
		}
		decoded, err := DecodeURIFragment(value)
		if err != nil {
			return nil, fmt.Errorf("reading %s from URL: %w", key, err)
		}
		out[key] = decoded
	}
	return out, nil
}

// decodeShareInput decodes the input parameter. BuildURL writes unpadded
// standard base64, but a link from elsewhere may be padded or use the URL-safe
// alphabet, so all four spellings are normalised to one before decoding. A
// character belonging to no alphabet is an error rather than something to drop
// silently, which would return input that is not what the link carried.
func decodeShareInput(s string) ([]byte, error) {
	s = strings.Join(strings.Fields(s), "")
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	s = strings.TrimRight(s, "=")
	b, err := base64.RawStdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("reading input from URL: %w", err)
	}
	return b, nil
}

// BuildURL constructs a CyberChef share URL for the given recipe and input,
// pointing at the instance named by base. The recipe is rendered in Chef
// format; the input is standard base64 (no padding), both then
// fragment-encoded.
func BuildURL(base string, r Recipe, input []byte) string {
	url := base + "#recipe=" + EncodeURIFragment(GeneratePrettyRecipe(r, false))
	if len(input) > 0 {
		enc := base64.RawStdEncoding.EncodeToString(input)
		url += "&input=" + EncodeURIFragment(enc)
	}
	return url
}
