package overlay

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDir_EmptyDirArg(t *testing.T) {
	got, err := LoadDir("")
	if err != nil {
		t.Fatalf("LoadDir(\"\"): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty map, got %v", got)
	}
}

func TestLoadDir_MissingDir(t *testing.T) {
	got, err := LoadDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadDir on missing dir: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty map, got %v", got)
	}
}

func TestLoadDir_ParsesMultipleModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "iam.yaml"), `commands:
  create-user:
    aliases: [adduser, new-user]
    short: "Create a user"
    long: "Long description for create-user."
    example: "myctl iam create-user --email a@b.c"
`)
	writeFile(t, filepath.Join(dir, "billing.yaml"), `commands:
  list-invoices:
    short: "List invoices"
`)
	writeFile(t, filepath.Join(dir, "README.md"), "should be ignored")

	got, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 modules, got %d: %v", len(got), got)
	}
	u := got["iam"]["create-user"]
	if u.Short != "Create a user" || u.Long == "" || u.Example == "" {
		t.Errorf("iam create-user override incomplete: %+v", u)
	}
	if len(u.Aliases) != 2 || u.Aliases[0] != "adduser" || u.Aliases[1] != "new-user" {
		t.Errorf("iam create-user aliases: %v", u.Aliases)
	}
	if got["billing"]["list-invoices"].Short != "List invoices" {
		t.Errorf("billing list-invoices: %+v", got["billing"]["list-invoices"])
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
