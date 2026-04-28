package runtime

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func captureStderr(t *testing.T) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		w.Close()
		os.Stderr = os.NewFile(2, "/dev/stderr")
	})
	return r
}

func readStderr(t *testing.T, r *os.File) string {
	t.Helper()
	os.Stderr.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

func TestDebugTransport_LogsJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	r := captureStderr(t)

	dt := &debugTransport{inner: http.DefaultTransport}
	body := strings.NewReader(`{"name":"test"}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", srv.URL+"/api", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	out := readStderr(t, r)
	if !strings.Contains(out, `{"name":"test"}`) {
		t.Errorf("stderr missing request body:\n%s", out)
	}
	if !strings.Contains(out, `{"result":"ok"}`) {
		t.Errorf("stderr missing response body:\n%s", out)
	}
	if !strings.Contains(out, "[body") {
		t.Errorf("stderr missing body size label:\n%s", out)
	}
}

func TestDebugTransport_TruncatesLargeBody(t *testing.T) {
	large := strings.Repeat("x", 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(large))
	}))
	defer srv.Close()

	r := captureStderr(t)

	dt := &debugTransport{inner: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	out := readStderr(t, r)
	if !strings.Contains(out, "showing first 4096") {
		t.Errorf("stderr missing truncation label:\n%s", out)
	}
}

func TestDebugTransport_SkipsBinaryBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4e, 0x47})
	}))
	defer srv.Close()

	r := captureStderr(t)

	dt := &debugTransport{inner: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()

	out := readStderr(t, r)
	if strings.Contains(out, "[body") {
		t.Errorf("stderr should not contain body dump for binary content:\n%s", out)
	}
}

func TestDebugTransport_PreservesResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":1}`))
	}))
	defer srv.Close()

	r := captureStderr(t)

	dt := &debugTransport{inner: http.DefaultTransport}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	resp, err := dt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	readStderr(t, r)

	if string(body) != `{"data":1}` {
		t.Errorf("response body = %q, want %q", string(body), `{"data":1}`)
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/xml", true},
		{"text/plain", true},
		{"text/html", true},
		{"application/x-www-form-urlencoded", true},
		{"image/png", false},
		{"application/octet-stream", false},
		{"", false},
		{"application/json; charset=utf-8", true},
	}
	for _, tc := range tests {
		if got := isTextContent(tc.ct); got != tc.want {
			t.Errorf("isTextContent(%q) = %v, want %v", tc.ct, got, tc.want)
		}
	}
}
