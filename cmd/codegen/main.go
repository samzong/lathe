package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/samzong/lathe/internal/codegen/backends/openapi3"
	"github.com/samzong/lathe/internal/codegen/backends/proto"
	"github.com/samzong/lathe/internal/codegen/backends/swagger"
	"github.com/samzong/lathe/internal/codegen/normalize"
	"github.com/samzong/lathe/internal/codegen/rawir"
	"github.com/samzong/lathe/internal/codegen/render"
	"github.com/samzong/lathe/internal/overlay"
	"github.com/samzong/lathe/internal/sourceconfig"
	"github.com/samzong/lathe/internal/specsync"
	"github.com/samzong/lathe/pkg/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return runWithOutput(args, os.Stderr)
}

func runWithOutput(args []string, output io.Writer) error {
	fs := flag.NewFlagSet("codegen", flag.ContinueOnError)
	fs.SetOutput(output)
	sourcesPath := fs.String("sources", "specs/sources.yaml", "sources.yaml path")
	manifestPath := fs.String("manifest", "cli.yaml", "cli.yaml path")
	cacheRoot := fs.String("cache", "", "cache root (default $LATHE_SPECS_CACHE or .cache)")
	overlayDir := fs.String("overlay", "", "directory containing <module>.yaml overlay files (optional)")
	skillRoot := fs.String("skill-root", "skills", "skill output root, or empty to disable skill generation")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := sourceconfig.Load(*sourcesPath)
	if err != nil {
		return err
	}

	overlays, err := overlay.LoadDir(*overlayDir)
	if err != nil {
		return err
	}

	root := *cacheRoot
	if root == "" {
		root = os.Getenv("LATHE_SPECS_CACHE")
	}
	if root == "" {
		root = ".cache"
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	syncRoot := filepath.Join(absRoot, specsync.SyncSubdir)

	var manifest *config.Manifest
	var skillDir string
	if *skillRoot != "" {
		data, err := os.ReadFile(*manifestPath)
		if err != nil {
			return err
		}
		manifest, err = config.Load(data)
		if err != nil {
			return err
		}
		skillDir, err = skillOutputDir(*skillRoot, manifest.CLI.Name)
		if err != nil {
			return err
		}
	}

	var names []string
	var skillModules []render.SkillModule
	for _, src := range cfg.Ordered() {
		syncDir := filepath.Join(syncRoot, src.Name)
		if err := specsync.VerifyState(syncDir, src.Name, src.Backend, src.PinnedTag); err != nil {
			return err
		}
		state, err := specsync.LoadState(syncDir)
		if err != nil {
			return err
		}

		mod, err := parseSource(src, syncDir)
		if err != nil {
			return err
		}

		specs := normalize.Normalize(mod)
		specs = render.MergeOverlay(specs, overlays[src.Name])
		if err := render.RenderModule(src.Name, specs, nil); err != nil {
			return err
		}
		names = append(names, src.Name)
		if manifest != nil {
			skillModules = append(skillModules, render.SkillModule{Source: src, State: state, Specs: specs})
		}
	}
	if err := render.RenderModulesGen(names); err != nil {
		return err
	}
	if manifest != nil {
		if err := render.RenderSkillDirectory(skillDir, manifest, skillModules); err != nil {
			return err
		}
	}
	return nil
}

func parseSource(src *sourceconfig.Source, syncDir string) (*rawir.RawModule, error) {
	switch src.Backend {
	case sourceconfig.BackendSwagger:
		return swagger.Parse(src, syncDir)
	case sourceconfig.BackendProto:
		return proto.Parse(src, syncDir)
	case sourceconfig.BackendOpenAPI3:
		return openapi3.Parse(src, syncDir)
	default:
		return nil, fmt.Errorf("source %q: unknown backend %q", src.Name, src.Backend)
	}
}

func skillOutputDir(root string, cliName string) (string, error) {
	clean := filepath.Clean(root)
	if root == "" || clean == "." || clean == ".." || clean == string(filepath.Separator) || hasParentTraversal(clean) {
		return "", fmt.Errorf("invalid skill root %q", root)
	}
	absRoot, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	if absRoot == absCWD {
		return "", fmt.Errorf("invalid skill root %q: must not be the current project root", root)
	}
	return filepath.Join(clean, render.SkillDirName(cliName)), nil
}

func hasParentTraversal(path string) bool {
	for _, part := range strings.Split(path, string(filepath.Separator)) {
		if part == ".." {
			return true
		}
	}
	return false
}
