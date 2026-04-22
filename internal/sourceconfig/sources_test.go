package sourceconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRef_Accepts(t *testing.T) {
	cases := []string{
		"v1.2.3",
		"v0.0.0-alpha",
		"release/2026.04",
		"1234567890abcdef1234567890abcdef12345678", // 40-hex SHA
	}
	for _, ref := range cases {
		if err := validateRef(ref); err != nil {
			t.Errorf("validateRef(%q) = %v, want nil", ref, err)
		}
	}
}

func TestValidateRef_Rejects(t *testing.T) {
	cases := []struct {
		name string
		ref  string
	}{
		{"head", "HEAD"},
		{"main", "main"},
		{"master", "master"},
		{"refs-heads", "refs/heads/main"},
		{"refs-remotes", "refs/remotes/origin/main"},
		{"leading-dash", "-rf"},
		{"contains-space", "v1 .0"},
		{"contains-tab", "v1\t0"},
		{"double-dot", "v1..0"},
		{"caret", "v1^0"},
		{"tilde", "v1~1"},
		{"colon", "v1:0"},
		{"question", "v?"},
		{"asterisk", "v*"},
		{"lbracket", "v["},
		{"backslash", "v\\x"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateRef(tc.ref)
			if err == nil {
				t.Fatalf("validateRef(%q) = nil, want floating-ref error", tc.ref)
			}
			if !strings.Contains(err.Error(), "floating ref") {
				t.Errorf("error message missing 'floating ref': %v", err)
			}
		})
	}
}

func TestLoad_RejectsFloatingPinnedTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sources.yaml")
	body := `sources:
  demo:
    repo_url: https://example.com/repo.git
    pinned_tag: main
    backend: swagger
    swagger:
      files: [api.json]
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("seed yaml: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatalf("Load accepted pinned_tag=main; want floating-ref rejection")
	}
	if !strings.Contains(err.Error(), "floating ref") {
		t.Errorf("error = %v, want to mention floating ref", err)
	}
}

func TestLoad_AcceptsImmutableTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sources.yaml")
	body := `sources:
  demo:
    repo_url: https://example.com/repo.git
    pinned_tag: v1.2.3
    backend: swagger
    swagger:
      files: [api.json]
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("seed yaml: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sources["demo"].PinnedTag != "v1.2.3" {
		t.Errorf("pinned_tag = %q, want v1.2.3", cfg.Sources["demo"].PinnedTag)
	}
}

func TestLoad_AcceptsFullSHA(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sources.yaml")
	body := `sources:
  demo:
    repo_url: https://example.com/repo.git
    pinned_tag: 1234567890abcdef1234567890abcdef12345678
    backend: proto
    proto:
      staging:
        - from: ./api
          to: api
      entries: [api/v1/service.proto]
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("seed yaml: %v", err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
}
