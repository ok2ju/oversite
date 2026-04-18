package auth

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

// debugTransport wraps an http.RoundTripper and logs request/response details
// to stderr. Activate by setting OVERSITE_DEBUG_HTTP=1.
type debugTransport struct {
	inner http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log request.
	dump, err := httputil.DumpRequestOut(req, false)
	if err != nil {
		log.Printf("[HTTP →] %s %s (dump error: %v)", req.Method, req.URL, err)
	} else {
		log.Printf("[HTTP →] %s %s\n%s", req.Method, req.URL, indent(dump))
	}

	resp, rtErr := t.inner.RoundTrip(req)
	elapsed := time.Since(start)

	if rtErr != nil {
		log.Printf("[HTTP ✗] %s %s — error after %s: %v", req.Method, req.URL, elapsed, rtErr)
		return nil, rtErr
	}

	// Log response. Skip body dump for large responses (e.g. demo downloads)
	// to avoid reading hundreds of MB into memory.
	dumpBody := resp.ContentLength < 1<<20 // <1 MB or unknown (-1)
	dump, err = httputil.DumpResponse(resp, dumpBody)
	if err != nil {
		log.Printf("[HTTP ←] %d %s (%s, dump error: %v)", resp.StatusCode, req.URL, elapsed, err)
	} else {
		const maxDump = 4096
		s := string(dump)
		if len(s) > maxDump {
			s = s[:maxDump] + fmt.Sprintf("\n... truncated (%d bytes total)", len(dump))
		}
		if !dumpBody {
			s += fmt.Sprintf("\n  [body omitted — Content-Length: %d]", resp.ContentLength)
		}
		log.Printf("[HTTP ←] %d %s (%s)\n%s", resp.StatusCode, req.URL, elapsed, indent([]byte(s)))
	}

	return resp, nil
}

func indent(b []byte) string {
	lines := strings.Split(strings.TrimRight(string(b), "\r\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

// newDebugTransport wraps the given transport (or http.DefaultTransport if nil)
// with request/response logging.
func newDebugTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &debugTransport{inner: base}
}

// newHTTPClient returns an *http.Client, optionally wrapped with debug logging.
func newHTTPClient(debug bool) *http.Client {
	if debug {
		return &http.Client{
			Transport: newDebugTransport(nil),
		}
	}
	return &http.Client{}
}

// NewDebugHTTPClient returns an *http.Client that logs requests/responses when
// OVERSITE_DEBUG_HTTP=1 is set. Exported for use outside the auth package
// (e.g., demo download client in app.go).
func NewDebugHTTPClient() *http.Client {
	return newHTTPClient(os.Getenv("OVERSITE_DEBUG_HTTP") == "1")
}
