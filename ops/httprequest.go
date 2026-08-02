package ops

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(HTTPRequest{})
}

// HTTPRequest makes an HTTP request and returns the response.
type HTTPRequest struct{}

// Meta returns the operation metadata.
func (HTTPRequest) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "HTTP request",
		Module:      "Default",
		Description: "Makes an HTTP request and returns the response. The body of the request is populated from the input. The Mode (CORS) argument is browser-specific and has no effect in this port.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_HTTP_header_fields#Request_fields",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (HTTPRequest) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Method", Type: core.ArgOption, Value: []string{"GET", "POST", "HEAD", "PUT", "PATCH", "DELETE", "CONNECT", "TRACE", "OPTIONS"}},
		{Name: "URL", Type: core.ArgString, Value: ""},
		{Name: "Headers", Type: core.ArgString, Value: ""},
		{Name: "Mode", Type: core.ArgOption, Value: []string{"Cross-Origin Resource Sharing", "No CORS (limited to HEAD, GET or POST)"}},
		{Name: "Show response metadata", Type: core.ArgBoolean, Value: false},
	}
}

// Run makes the HTTP request. The Mode argument (browser CORS) is accepted
// for parity but has no effect here.
func (HTTPRequest) Run(in *core.Dish, args []any) (*core.Dish, error) {
	method := args[0].(string)
	url := args[1].(string)
	headersText := args[2].(string)
	showMeta := args[4].(bool)

	if len(url) == 0 {
		return core.NewDish([]byte(""), core.TypeString), nil
	}

	var body io.Reader
	if method != "GET" && method != "HEAD" {
		body = strings.NewReader(in.String())
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for line := range strings.SplitSeq(headersText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		split := strings.Split(line, ":")
		if len(split) != 2 {
			return nil, fmt.Errorf("could not parse header in line: %s", line)
		}
		req.Header.Set(strings.TrimSpace(split[0]), strings.TrimSpace(split[1]))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w\n\nThis error could be caused by one of the following:\n"+
			" - An invalid URL\n"+
			" - Making a request to an insecure resource (HTTP) from a secure source (HTTPS)\n"+
			" - Making a cross-origin request to a server which does not support CORS", err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if !showMeta {
		return core.NewDish(b, core.TypeString), nil
	}

	keys := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var hb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&hb, "    %s: %s\n", strings.ToLower(k), strings.Join(resp.Header[k], ", "))
	}
	out := "####\n  Status: " + resp.Status + "\n  Exposed headers:\n" + hb.String() + "####\n\n" + string(b)
	return core.NewDish([]byte(out), core.TypeString), nil
}
