package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(DNSOverHTTPS{})
}

// DNSOverHTTPS resolves a domain via a DNS-over-HTTPS resolver.
type DNSOverHTTPS struct{}

// Meta returns the operation metadata.
func (DNSOverHTTPS) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "DNS over HTTPS",
		Module:      "Default",
		Description: "Takes a single domain name and performs a DNS lookup using DNS over HTTPS. By default, Cloudflare and Google DNS over HTTPS services are supported.",
		InfoURL:     "https://wikipedia.org/wiki/DNS_over_HTTPS",
		InputType:   core.TypeString,
		OutputType:  core.TypeJSON,
	}
}

// Args returns the argument definitions.
func (DNSOverHTTPS) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Resolver", Type: core.ArgEditableOption, Value: "https://dns.google.com/resolve"},
		{Name: "Request Type", Type: core.ArgOption, Value: []string{"A", "AAAA", "ANAME", "CERT", "CNAME", "DNSKEY", "HTTPS", "IPSECKEY", "LOC", "MX", "NS", "OPENPGPKEY", "PTR", "RRSIG", "SIG", "SOA", "SPF", "SRV", "SSHFP", "TA", "TXT", "URI", "ANY"}},
		{Name: "Answer Data Only", Type: core.ArgBoolean, Value: false},
		{Name: "Disable DNSSEC validation", Type: core.ArgBoolean, Value: false},
	}
}

// Run performs the DNS-over-HTTPS lookup. Ported from CyberChef DNSOverHTTPS.mjs.
func (DNSOverHTTPS) Run(in *core.Dish, args []any) (*core.Dish, error) {
	resolver := args[0].(string)
	requestType := args[1].(string)
	justAnswer := args[2].(bool)
	dnssec := args[3].(bool)

	u, err := url.Parse(resolver)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid resolver URL %q", resolver)
	}
	q := u.Query()
	q.Set("name", in.String())
	q.Set("type", requestType)
	q.Set("cd", strconv.FormatBool(dnssec))
	u.RawQuery = q.Encode()

	// Build the request straight from the already-parsed URL rather than
	// re-serialising and re-parsing it via http.NewRequest.
	req := &http.Request{Method: http.MethodGet, URL: u, Header: http.Header{}}
	req.Header.Set("Accept", "application/dns-json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to %s: %w", u.String(), err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if !justAnswer {
		return core.NewDish(body, core.TypeJSON), nil
	}

	var parsed struct {
		Answer []struct {
			Data string `json:"data"`
		} `json:"Answer"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("error making request to %s: %w", u.String(), err)
	}
	values := []string{}
	for _, a := range parsed.Answer {
		values = append(values, a.Data)
	}
	out, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeJSON), nil
}
