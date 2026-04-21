package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samzong/lathe/internal/codegen/backends/proto"
	"github.com/samzong/lathe/internal/codegen/backends/swagger"
	"github.com/samzong/lathe/internal/codegen/normalize"
	"github.com/samzong/lathe/internal/codegen/rawir"
	"github.com/samzong/lathe/internal/codegen/render"
	"github.com/samzong/lathe/internal/overlay"
	"github.com/samzong/lathe/internal/sourceconfig"
	"github.com/samzong/lathe/internal/specsync"
)

func main() {
	sourcesPath := flag.String("sources", "specs/sources.yaml", "sources.yaml path")
	cacheRoot := flag.String("cache", "", "cache root (default $LATHE_SPECS_CACHE or .cache)")
	overlayDir := flag.String("overlay", "", "directory containing <module>.yaml overlay files (optional)")
	flag.Parse()

	cfg, err := sourceconfig.Load(*sourcesPath)
	must(err)

	overlays, err := overlay.LoadDir(*overlayDir)
	must(err)

	root := *cacheRoot
	if root == "" {
		root = os.Getenv("LATHE_SPECS_CACHE")
	}
	if root == "" {
		root = ".cache"
	}
	absRoot, err := filepath.Abs(root)
	must(err)
	syncRoot := filepath.Join(absRoot, specsync.SyncSubdir)

	var names []string
	for _, src := range cfg.Ordered() {
		syncDir := filepath.Join(syncRoot, src.Name)
		must(specsync.VerifyState(syncDir, src.Name, src.Backend, src.PinnedTag))

		mod, err := parseSource(src, syncDir)
		must(err)

		specs := normalize.Normalize(mod)
		must(render.RenderModule(src.Name, specs, overlays[src.Name]))
		names = append(names, src.Name)
	}
	must(render.RenderModulesGen(names))
}

func parseSource(src *sourceconfig.Source, syncDir string) (*rawir.RawModule, error) {
	switch src.Backend {
	case sourceconfig.BackendSwagger:
		return swagger.Parse(src, syncDir)
	case sourceconfig.BackendProto:
		return proto.Parse(src, syncDir)
	default:
		return nil, fmt.Errorf("source %q: unknown backend %q", src.Name, src.Backend)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
