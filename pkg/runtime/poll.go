package runtime

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	DefaultPollTimeout = 5 * time.Minute
	pollInitBackoff    = 1 * time.Second
	pollMaxBackoff     = 30 * time.Second
)

func PollUntilDone(ctx context.Context, hostname, location string, opts ClientOptions, timeout time.Duration) ([]byte, error) {
	if timeout <= 0 {
		timeout = DefaultPollTimeout
	}
	deadline := time.Now().Add(timeout)
	backoff := pollInitBackoff

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("polling timed out after %s", timeout)
		}

		r, err := DoRawFull(ctx, hostname, "GET", location, nil, opts)
		if err != nil {
			return nil, err
		}

		if r.StatusCode != http.StatusAccepted {
			return r.Body, nil
		}

		if loc := r.Header.Get("Location"); loc != "" {
			location = loc
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > pollMaxBackoff {
			backoff = pollMaxBackoff
		}
	}
}
