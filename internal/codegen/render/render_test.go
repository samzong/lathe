package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samzong/lathe/internal/overlay"
	"github.com/samzong/lathe/pkg/runtime"
)

func TestRenderModule_AppliesOverlay(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("go.mod", []byte("module example.com/fake\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	specs := []runtime.CommandSpec{
		{Group: "Addon", Use: "install-addon", Short: "raw short", Method: "POST", PathTpl: "/api/v1/addon", RequestBody: &runtime.RequestBody{Required: true}},
		{Group: "Addon", Use: "untouched", Short: "untouched short", Method: "GET", PathTpl: "/api/v1/x"},
	}
	overrides := map[string]overlay.Override{
		"install-addon": {
			Aliases: []string{"addon-install"},
			Short:   "OVERLAY SHORT",
			Long:    "OVERLAY LONG DESC",
			Example: "myctl demo install-addon --name foo",
		},
	}

	if err := RenderModule("demo", specs, overrides); err != nil {
		t.Fatalf("RenderModule: %v", err)
	}
	out, err := os.ReadFile(filepath.Join("internal/generated/demo/demo_gen.go"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		`"OVERLAY SHORT"`,
		`"OVERLAY LONG DESC"`,
		`"myctl demo install-addon --name foo"`,
		`"addon-install"`,
		`"untouched short"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if strings.Contains(got, `"raw short"`) {
		t.Errorf("overlay did not replace Short; raw value leaked into output")
	}
}

func TestRenderModule_NilOverrides(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("go.mod", []byte("module example.com/fake\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	specs := []runtime.CommandSpec{
		{Group: "Addon", Use: "install-addon", Short: "raw short", Method: "POST", PathTpl: "/x"},
	}
	if err := RenderModule("demo", specs, nil); err != nil {
		t.Fatalf("RenderModule nil overrides: %v", err)
	}
	out, err := os.ReadFile(filepath.Join("internal/generated/demo/demo_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"raw short"`) {
		t.Errorf("expected raw short preserved when overrides is nil")
	}
}
