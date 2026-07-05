package ops

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(ParseURI{})
}

// padEndSpace right-pads s with spaces to at least n characters.
func padEndSpace(s string, n int) string {
	if len(s) < n {
		return s + strings.Repeat(" ", n-len(s))
	}
	return s
}

// ParseURI breaks a URI into its component parts.
type ParseURI struct{}

// Meta returns the operation metadata.
func (ParseURI) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse URI",
		Module:      "URL",
		Description: "Pretty prints URI (Uniform Resource Identifier) strings for ease of reading. Particularly useful for URLs and Data URIs.",
		InfoURL:     "https://wikipedia.org/wiki/Uniform_Resource_Identifier",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseURI) Args() []core.ArgDef { return nil }

// Run parses the URI. Ported from CyberChef ParseURI.mjs (Node url -> net/url).
func (ParseURI) Run(in *core.Dish, args []any) (*core.Dish, error) {
	u, err := url.Parse(strings.TrimSpace(in.String()))
	if err != nil {
		return nil, err
	}

	var out strings.Builder
	if u.Scheme != "" {
		fmt.Fprintf(&out, "Protocol:\t%s:\n", u.Scheme)
	}
	if u.User != nil {
		fmt.Fprintf(&out, "Auth:\t\t%s\n", u.User.String())
	}
	if h := u.Hostname(); h != "" {
		fmt.Fprintf(&out, "Hostname:\t%s\n", h)
	}
	if p := u.Port(); p != "" {
		fmt.Fprintf(&out, "Port:\t\t%s\n", p)
	}
	// Node defaults the path to "/" when a host is present but no path is given.
	pathname := u.Path
	if pathname == "" && u.Host != "" {
		pathname = "/"
	}
	if pathname != "" {
		fmt.Fprintf(&out, "Path name:\t%s\n", pathname)
	}

	// Node's url.parse(..., true) always yields a query object, so "Arguments:"
	// is always printed. Parse RawQuery manually to preserve insertion order.
	var order []string
	values := map[string][]string{}
	for part := range strings.SplitSeq(u.RawQuery, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		k, _ = url.QueryUnescape(k)
		v, _ = url.QueryUnescape(v)
		if _, ok := values[k]; !ok {
			order = append(order, k)
		}
		values[k] = append(values[k], v)
	}
	padding := 0
	for _, k := range order {
		if len(k) > padding {
			padding = len(k)
		}
	}
	out.WriteString("Arguments:\n")
	for _, k := range order {
		out.WriteString("\t" + padEndSpace(k, padding))
		if val := strings.Join(values[k], ","); val != "" {
			out.WriteString(" = " + val)
		}
		out.WriteString("\n")
	}

	if u.Fragment != "" {
		fmt.Fprintf(&out, "Hash:\t\t#%s\n", u.EscapedFragment())
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}
