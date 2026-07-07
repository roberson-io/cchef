package ops

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// httpTestServer echoes request details so tests can assert method, body and
// headers were sent correctly.
func httpTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Test-Header", "cchef")
		_, _ = fmt.Fprintf(w, "method=%s body=%s xcustom=%s", r.Method, body, r.Header.Get("X-Custom"))
	})
	mux.HandleFunc("/status418", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = fmt.Fprint(w, "teapot")
	})
	// Declares more bytes than it sends, then hijacks and closes the connection so
	// the client's body read fails with an unexpected EOF.
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
	return s
}

func TestHTTPRequest(t *testing.T) {
	s := httpTestServer(t)
	const cors = "Cross-Origin Resource Sharing"

	// GET returns the response body; body arg is ignored for GET.
	if out, err := runOp(t, "HTTP request", "ignored", "GET", s.URL+"/echo", "", cors, false); err != nil ||
		out != "method=GET body= xcustom=" {
		t.Errorf("GET: %q err %v", out, err)
	}
	// POST sends the input as the request body.
	if out, err := runOp(t, "HTTP request", "hello", "POST", s.URL+"/echo", "", cors, false); err != nil ||
		out != "method=POST body=hello xcustom=" {
		t.Errorf("POST: %q err %v", out, err)
	}
	// Custom headers are parsed and forwarded.
	if out, err := runOp(t, "HTTP request", "", "GET", s.URL+"/echo", "X-Custom: abc", cors, false); err != nil ||
		out != "method=GET body= xcustom=abc" {
		t.Errorf("headers: %q err %v", out, err)
	}
	// Show response metadata prepends status and exposed headers.
	out, err := runOp(t, "HTTP request", "", "GET", s.URL+"/status418", "", cors, true)
	if err != nil || !strings.Contains(out, "Status: 418") || !strings.HasSuffix(out, "####\n\nteapot") {
		t.Errorf("metadata: %q err %v", out, err)
	}
	// Empty URL returns an empty string, no request made.
	if o, err := runOp(t, "HTTP request", "x", "GET", "", "", cors, false); err != nil || o != "" {
		t.Errorf("empty URL: %q err %v", o, err)
	}
	// A malformed header line (no single colon-separated pair) errors.
	if _, err := runOp(t, "HTTP request", "", "GET", s.URL+"/echo", "no-colon-here", cors, false); err == nil {
		t.Error("malformed header: expected error")
	}
	// An unreachable/invalid URL errors.
	if _, err := runOp(t, "HTTP request", "", "GET", "http://127.0.0.1:1/nope", "", cors, false); err == nil {
		t.Error("bad URL: expected error")
	}
	// A URL with a control character fails http.NewRequest before any network I/O.
	if _, err := runOp(t, "HTTP request", "", "GET", "http://foo\x01bar", "", cors, false); err == nil {
		t.Error("control-char URL: expected a NewRequest error")
	}
	// A response body that ends early (declared Content-Length exceeds the bytes
	// sent) fails the body read.
	if _, err := runOp(t, "HTTP request", "", "GET", s.URL+"/truncate", "", cors, false); err == nil {
		t.Error("truncated response: expected a read error")
	}
}
