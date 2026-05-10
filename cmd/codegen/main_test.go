package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_GeneratesSkillDirectoryByDefault(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	seedCodegenProject(t, true)

	if err := run([]string{"-sources", "specs/sources.yaml", "-cache", ".cache"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	for _, path := range []string{
		"internal/generated/acme/acme_gen.go",
		"internal/generated/modules_gen.go",
		"skills/acmectl/SKILL.md",
		"skills/acmectl/agents/openai.yaml",
		"skills/acmectl/references/catalog.md",
		"skills/acmectl/references/modules/acme.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}

	skill := readCodegenFile(t, "skills/acmectl/SKILL.md")
	if !strings.Contains(skill, "acmectl search \"<intent>\" --json") {
		t.Fatalf("skill missing search workflow:\n%s", skill)
	}
	module := readCodegenFile(t, "skills/acmectl/references/modules/acme.md")
	if !strings.Contains(module, "Resolved SHA: `0000000000000000000000000000000000000000`") {
		t.Fatalf("module reference missing resolved SHA:\n%s", module)
	}
}

func TestRun_SkillRootEmptyDisablesSkillGeneration(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	seedCodegenProject(t, false)

	if err := run([]string{"-sources", "specs/sources.yaml", "-cache", ".cache", "-skill-root", ""}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat("skills"); !os.IsNotExist(err) {
		t.Fatalf("skills directory should not exist, stat err = %v", err)
	}
}

func TestRun_MissingManifestFailsWhenSkillEnabled(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	seedCodegenProject(t, false)

	err := run([]string{"-sources", "specs/sources.yaml", "-cache", ".cache"})
	if err == nil {
		t.Fatal("expected missing manifest error")
	}
	if !strings.Contains(err.Error(), "cli.yaml") {
		t.Fatalf("error should mention cli.yaml, got %v", err)
	}
}

func TestRunWithOutput_HelpPrintsUsage(t *testing.T) {
	var out bytes.Buffer
	err := runWithOutput([]string{"-h"}, &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
	got := out.String()
	for _, want := range []string{"Usage of codegen:", "-manifest", "-skill-root"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}

func TestRun_RejectsUnsafeSkillRootBeforeDeleting(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	seedCodegenProject(t, true)
	writeCodegenFile(t, "cli.yaml", "cli:\n  name: internal\n  short: Internal CLI\n")
	writeCodegenFile(t, "internal/sentinel.txt", "keep")

	err := run([]string{"-sources", "specs/sources.yaml", "-cache", ".cache", "-skill-root", "."})
	if err == nil {
		t.Fatal("expected unsafe skill root error")
	}
	if !strings.Contains(err.Error(), "invalid skill root") {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readCodegenFile(t, "internal/sentinel.txt"); got != "keep" {
		t.Fatalf("sentinel was changed: %q", got)
	}
	if _, err := os.Stat("internal/generated"); !os.IsNotExist(err) {
		t.Fatalf("codegen should fail before writing generated code, stat err = %v", err)
	}
}

func TestSkillOutputDirRejectsUnsafeRoots(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	for _, root := range []string{".", "..", string(filepath.Separator), "../skills"} {
		if _, err := skillOutputDir(root, "acmectl"); err == nil {
			t.Fatalf("expected %q to be rejected", root)
		}
	}
}

func seedCodegenProject(t *testing.T, withManifest bool) {
	t.Helper()
	writeCodegenFile(t, "go.mod", "module example.com/fake\n\ngo 1.25\n")
	if withManifest {
		writeCodegenFile(t, "cli.yaml", "cli:\n  name: acmectl\n  short: Acme CLI\n")
	}
	writeCodegenFile(t, "specs/sources.yaml", `sources:
  acme:
    repo_url: https://example.com/acme.git
    pinned_tag: v1.0.0
    backend: openapi3
    openapi3:
      files: [openapi.yaml]
`)
	writeCodegenFile(t, ".cache/specs-sync/acme/sync-state.yaml", `source: acme
backend: openapi3
synced_from: v1.0.0
resolved_sha: "0000000000000000000000000000000000000000"
`)
	writeCodegenFile(t, ".cache/specs-sync/acme/openapi.yaml", `openapi: "3.0.3"
paths:
  /users:
    get:
      operationId: Users_List
      tags: [Users]
      summary: List users
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  items:
                    type: array
                    items:
                      type: object
`)
}

func writeCodegenFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readCodegenFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
