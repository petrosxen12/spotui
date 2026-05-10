package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/petrosxen/spotui/internal/ui"
)

func main() {
	outDir := flag.String("out", "docs/qa/tui-review", "output directory for generated QA review assets")
	flag.Parse()

	if err := ui.WriteQAReviewBundle(*outDir); err != nil {
		fmt.Fprintf(os.Stderr, "spotui-qareview: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote TUI QA review bundle to %s\n", *outDir)
}
