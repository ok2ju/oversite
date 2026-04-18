package logging

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewTransport_DumpsRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer server.Close()

	var buf bytes.Buffer
	client := &http.Client{Transport: NewTransport(http.DefaultTransport, &buf)}

	resp, err := client.Get(server.URL + "/ping")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	got := buf.String()
	if !strings.Contains(got, "[HTTP →] GET") {
		t.Errorf("request marker missing: %s", got)
	}
	if !strings.Contains(got, "[HTTP ←] 200") {
		t.Errorf("response marker missing: %s", got)
	}
	if !strings.Contains(got, `{"hello":"world"}`) {
		t.Errorf("response body missing: %s", got)
	}
}

func TestNewTransport_SkipsLargeBody(t *testing.T) {
	// Advertise a >1MB Content-Length so the transport skips the body dump.
	// We don't actually have to send that many bytes because DumpResponse
	// only inspects Content-Length when deciding to include the body.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "2000000")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		// Write exactly 2M bytes of zeros.
		chunk := make([]byte, 64*1024)
		for written := 0; written < 2000000; {
			n := len(chunk)
			if 2000000-written < n {
				n = 2000000 - written
			}
			if _, err := w.Write(chunk[:n]); err != nil {
				return
			}
			written += n
		}
	}))
	defer server.Close()

	var buf bytes.Buffer
	client := &http.Client{Transport: NewTransport(http.DefaultTransport, &buf)}

	resp, err := client.Get(server.URL + "/big")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	got := buf.String()
	if !strings.Contains(got, "body omitted") {
		t.Errorf("expected 'body omitted' marker, got:\n%s", got)
	}
}
