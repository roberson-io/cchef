package core

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// cyberChefBaseURL is the public CyberChef instance used for share links.
const cyberChefBaseURL = "https://gchq.github.io/CyberChef/"

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

// BuildURL constructs a CyberChef share URL for the given recipe and input. The
// recipe is rendered in Chef format; the input is standard base64 (no padding),
// both then fragment-encoded.
func BuildURL(r Recipe, input []byte) string {
	url := cyberChefBaseURL + "#recipe=" + EncodeURIFragment(GeneratePrettyRecipe(r, false))
	if len(input) > 0 {
		enc := base64.RawStdEncoding.EncodeToString(input)
		url += "&input=" + EncodeURIFragment(enc)
	}
	return url
}
