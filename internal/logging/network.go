package logging

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// OpenNetworkLog opens {dir}/network.txt as a rotating log file using the
// same policy as the errors log (5MB, 3 backups). Callers are responsible
// for closing the returned logger on shutdown.
func OpenNetworkLog(dir string) (*lumberjack.Logger, error) {
	return &lumberjack.Logger{
		Filename:   filepath.Join(dir, networkFileName),
		MaxSize:    defaultMaxSizeMB,
		MaxBackups: defaultMaxBackups,
		Compress:   false,
	}, nil
}

// NewTransport wraps base with a RoundTripper that writes full request and
// response dumps to w. Bodies over 1MB are omitted to avoid loading demo
// downloads into memory; individual dumps are truncated at 4KB.
//
// If base is nil, http.DefaultTransport is used.
func NewTransport(base http.RoundTripper, w io.Writer) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if w == nil {
		w = io.Discard
	}
	return &dumpTransport{inner: base, w: w}
}

type dumpTransport struct {
	inner http.RoundTripper
	w     io.Writer
}

func (t *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	if dump, err := httputil.DumpRequestOut(req, false); err != nil {
		_, _ = fmt.Fprintf(t.w, "[HTTP →] %s %s (dump error: %v)\n", req.Method, req.URL, err)
	} else {
		_, _ = fmt.Fprintf(t.w, "[HTTP →] %s %s\n%s\n", req.Method, req.URL, indent(dump))
	}

	resp, rtErr := t.inner.RoundTrip(req)
	elapsed := time.Since(start)

	if rtErr != nil {
		_, _ = fmt.Fprintf(t.w, "[HTTP ✗] %s %s — error after %s: %v\n", req.Method, req.URL, elapsed, rtErr)
		return nil, rtErr
	}

	// Skip body dump for large responses (e.g. demo downloads).
	dumpBody := resp.ContentLength < 1<<20 // <1 MB or unknown (-1)
	dump, err := httputil.DumpResponse(resp, dumpBody)
	if err != nil {
		_, _ = fmt.Fprintf(t.w, "[HTTP ←] %d %s (%s, dump error: %v)\n", resp.StatusCode, req.URL, elapsed, err)
		return resp, nil
	}

	const maxDump = 4096
	s := string(dump)
	if len(s) > maxDump {
		s = s[:maxDump] + fmt.Sprintf("\n... truncated (%d bytes total)", len(dump))
	}
	if !dumpBody {
		s += fmt.Sprintf("\n  [body omitted — Content-Length: %d]", resp.ContentLength)
	}
	_, _ = fmt.Fprintf(t.w, "[HTTP ←] %d %s (%s)\n%s\n", resp.StatusCode, req.URL, elapsed, indent([]byte(s)))

	return resp, nil
}

func indent(b []byte) string {
	lines := strings.Split(strings.TrimRight(string(b), "\r\n"), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}
