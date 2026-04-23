package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestPollUntilDone_ImmediateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "done"})
	}))
	defer srv.Close()

	data, err := PollUntilDone(context.Background(), srv.URL, "/status", ClientOptions{Timeout: 5 * time.Second}, 10*time.Second)
	if err != nil {
		t.Fatalf("PollUntilDone: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "done" {
		t.Errorf("got status %q, want done", result["status"])
	}
}

func TestPollUntilDone_EventualSuccess(t *testing.T) {
	var call int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&call, 1)
		if n < 3 {
			w.Header().Set("Location", "/status")
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte(`{"status":"pending"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "done"})
	}))
	defer srv.Close()

	data, err := PollUntilDone(context.Background(), srv.URL, "/status", ClientOptions{Timeout: 5 * time.Second}, 30*time.Second)
	if err != nil {
		t.Fatalf("PollUntilDone: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["status"] != "done" {
		t.Errorf("got status %q, want done", result["status"])
	}
	if atomic.LoadInt32(&call) != 3 {
		t.Errorf("made %d requests, want 3", atomic.LoadInt32(&call))
	}
}

func TestPollUntilDone_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/status")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"pending"}`))
	}))
	defer srv.Close()

	_, err := PollUntilDone(context.Background(), srv.URL, "/status", ClientOptions{Timeout: 5 * time.Second}, 2*time.Second)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPollUntilDone_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/status")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"pending"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	_, err := PollUntilDone(ctx, srv.URL, "/status", ClientOptions{Timeout: 5 * time.Second}, 30*time.Second)
	if err == nil {
		t.Fatal("expected context error")
	}
}
