package ops

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDNSOverHTTPS(t *testing.T) {
	var lastQuery string
	var lastAccept string
	mux := http.NewServeMux()
	mux.HandleFunc("/dns", func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		lastAccept = r.Header.Get("Accept")
		_, _ = fmt.Fprint(w, `{"Status":0,"Answer":[{"name":"example.com","type":1,"data":"93.184.216.34"},{"name":"example.com","type":1,"data":"1.2.3.4"}]}`)
	})
	mux.HandleFunc("/noanswer", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"Status":3}`)
	})
	mux.HandleFunc("/badjson", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `not json`)
	})
	// Declares more bytes than it sends, then closes, so the client body read fails.
	mux.HandleFunc("/truncate", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
		if hj, ok := w.(http.Hijacker); ok {
			if conn, _, err := hj.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
	})
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)

	// Full JSON response passthrough, and request construction (name/type/cd, Accept header).
	out, err := runOp(t, "DNS over HTTPS", "example.com", s.URL+"/dns", "A", false, false)
	if err != nil || !strings.Contains(out, "93.184.216.34") || !strings.Contains(out, `"Status":0`) {
		t.Errorf("full: %q err %v", out, err)
	}
	if !strings.Contains(lastQuery, "name=example.com") || !strings.Contains(lastQuery, "type=A") || !strings.Contains(lastQuery, "cd=false") {
		t.Errorf("query params: %q", lastQuery)
	}
	if lastAccept != "application/dns-json" {
		t.Errorf("accept header: %q", lastAccept)
	}

	// DNSSEC toggle sets cd=true.
	if _, err := runOp(t, "DNS over HTTPS", "example.com", s.URL+"/dns", "AAAA", false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(lastQuery, "cd=true") || !strings.Contains(lastQuery, "type=AAAA") {
		t.Errorf("dnssec query: %q", lastQuery)
	}

	// Answer-data-only extracts the data values as a JSON array.
	if out, err := runOp(t, "DNS over HTTPS", "example.com", s.URL+"/dns", "A", true, false); err != nil ||
		out != `["93.184.216.34","1.2.3.4"]` {
		t.Errorf("just answer: %q err %v", out, err)
	}
	// Answer-data-only with no Answer field yields an empty array.
	if out, err := runOp(t, "DNS over HTTPS", "example.com", s.URL+"/noanswer", "A", true, false); err != nil || out != `[]` {
		t.Errorf("no answer: %q err %v", out, err)
	}

	// Invalid resolver URL errors.
	if _, err := runOp(t, "DNS over HTTPS", "example.com", "://not a url", "A", false, false); err == nil {
		t.Error("invalid resolver: expected error")
	}
	// Unreachable resolver errors.
	if _, err := runOp(t, "DNS over HTTPS", "example.com", "http://127.0.0.1:1/nope", "A", false, false); err == nil {
		t.Error("unreachable resolver: expected error")
	}
	// Answer-data-only with a non-JSON response body errors on unmarshal.
	if _, err := runOp(t, "DNS over HTTPS", "example.com", s.URL+"/badjson", "A", true, false); err == nil {
		t.Error("bad JSON response: expected an unmarshal error")
	}
	// A response body that ends early fails the body read.
	if _, err := runOp(t, "DNS over HTTPS", "example.com", s.URL+"/truncate", "A", false, false); err == nil {
		t.Error("truncated response: expected a read error")
	}
}
