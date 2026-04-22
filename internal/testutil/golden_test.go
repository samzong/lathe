package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeTB struct {
	helpers int
	errors  []string
	fatals  []string
}

func (f *fakeTB) Helper() { f.helpers++ }
func (f *fakeTB) Errorf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}
func (f *fakeTB) Fatalf(format string, args ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
}

func TestAssertGolden_WritesWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "out.json")

	tb := &fakeTB{}
	AssertGolden(tb, path, []byte("hello\n"))
	if len(tb.errors) != 0 || len(tb.fatals) != 0 {
		t.Fatalf("unexpected failures: errors=%v fatals=%v", tb.errors, tb.fatals)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != "hello\n" {
		t.Errorf("file content = %q, want %q", got, "hello\n")
	}
}

func TestAssertGolden_PassesWhenEqual(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "golden.json")
	if err := os.WriteFile(path, []byte("same\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tb := &fakeTB{}
	AssertGolden(tb, path, []byte("same\n"))
	if len(tb.errors) != 0 || len(tb.fatals) != 0 {
		t.Errorf("unexpected failures: errors=%v fatals=%v", tb.errors, tb.fatals)
	}
}

func TestAssertGolden_FailsWhenDifferent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "golden.json")
	if err := os.WriteFile(path, []byte("want\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tb := &fakeTB{}
	AssertGolden(tb, path, []byte("got\n"))

	if len(tb.errors) != 1 {
		t.Fatalf("want 1 Errorf call, got %d: %v", len(tb.errors), tb.errors)
	}
	msg := tb.errors[0]
	if !strings.Contains(msg, "golden mismatch") {
		t.Errorf("error message missing 'golden mismatch': %q", msg)
	}
	if !strings.Contains(msg, "- want") || !strings.Contains(msg, "+ got") {
		t.Errorf("error message missing diff markers: %q", msg)
	}
}

func TestLineDiff_ShowsAddedAndRemoved(t *testing.T) {
	d := lineDiff([]byte("a\nb\nc\n"), []byte("a\nB\nc\n"))
	for _, w := range []string{"- b", "+ B"} {
		if !strings.Contains(d, w+"\n") {
			t.Errorf("diff missing line %q; full diff:\n%s", w, d)
		}
	}
}
