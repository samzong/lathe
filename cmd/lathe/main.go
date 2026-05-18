package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/samzong/lathe/internal/lathecmd"
)

func main() {
	if err := lathecmd.Run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
