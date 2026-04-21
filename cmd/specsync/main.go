package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samzong/lathe/internal/sourceconfig"
	"github.com/samzong/lathe/internal/specsync"
)

func main() {
	sourcesPath := flag.String("sources", "specs/sources.yaml", "sources.yaml path")
	cacheRoot := flag.String("cache", "", "cache root (default $LATHE_SPECS_CACHE or .cache)")
	filter := flag.String("source", "", "sync only this source")
	flag.Parse()

	cfg, err := sourceconfig.Load(*sourcesPath)
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

	must(specsync.Sync(cfg, specsync.Options{
		CacheRoot: absRoot,
		Filter:    *filter,
	}))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
