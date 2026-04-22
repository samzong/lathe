package runtime

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
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
	fmt.Fprintln(os.Stderr)

	return resp, nil
}
