package runtime

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	maxDebugReqBody  = 1024
	maxDebugRespBody = 4096
)

type debugTransport struct {
	inner http.RoundTripper
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Fprintf(os.Stderr, "> %s %s\n", req.Method, req.URL)
	for k, vs := range req.Header {
		v := strings.Join(vs, ", ")
		if strings.EqualFold(k, "authorization") {
			v = "***"
		}
		fmt.Fprintf(os.Stderr, "> %s: %s\n", k, v)
	}
	if req.Body != nil && isTextContent(req.Header.Get("Content-Type")) {
		body, restored := peekBody(req.Body, maxDebugReqBody)
		req.Body = restored
		dumpBody(os.Stderr, ">", body, maxDebugReqBody)
	}
	fmt.Fprintln(os.Stderr)

	start := time.Now()
	resp, err := d.inner.RoundTrip(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "< error: %v (%s)\n\n", err, elapsed)
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "< %s (%s)\n", resp.Status, elapsed)
	for k, vs := range resp.Header {
		fmt.Fprintf(os.Stderr, "< %s: %s\n", k, strings.Join(vs, ", "))
	}
	if isTextContent(resp.Header.Get("Content-Type")) {
		body, restored := peekBody(resp.Body, maxDebugRespBody)
		resp.Body = restored
		dumpBody(os.Stderr, "<", body, maxDebugRespBody)
	}
	fmt.Fprintln(os.Stderr)

	return resp, nil
}

type bodyReader struct {
	io.Reader
	io.Closer
}

func isTextContent(ct string) bool {
	if ct == "" {
		return false
	}
	mt, _, _ := mime.ParseMediaType(ct)
	switch {
	case strings.HasPrefix(mt, "text/"):
		return true
	case mt == "application/json", mt == "application/xml", mt == "application/x-www-form-urlencoded":
		return true
	}
	return false
}

func peekBody(body io.ReadCloser, max int) ([]byte, io.ReadCloser) {
	peeked, err := io.ReadAll(io.LimitReader(body, int64(max)+1))
	restored := bodyReader{Reader: io.MultiReader(bytes.NewReader(peeked), body), Closer: body}
	if err != nil {
		return nil, restored
	}
	return peeked, restored
}

func dumpBody(w io.Writer, prefix string, body []byte, max int) {
	if len(body) == 0 {
		return
	}
	if len(body) > max {
		fmt.Fprintf(w, "%s [body at least %d bytes, showing first %d]\n", prefix, len(body), max)
		fmt.Fprintln(w, string(body[:max]))
	} else {
		fmt.Fprintf(w, "%s [body %d bytes]\n", prefix, len(body))
		fmt.Fprintln(w, string(body))
	}
}
