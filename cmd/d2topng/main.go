// Command d2topng renders a D2 diagram to a PNG file without any browser or
// SVG step: D2 is used purely as a parser/layout library, and drawing is done
// natively in Go.
package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"os"

	"github.com/macsplit/d2topng/internal/render"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "d2topng:", err)
		os.Exit(1)
	}
}

func run() error {
	scale := flag.Float64("scale", 1, "output resolution multiplier (e.g. 2 for double-resolution PNG)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: d2topng [-scale N] <input.d2> <output.png>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	inPath, outPath := flag.Arg(0), flag.Arg(1)

	src, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inPath, err)
	}

	diagram, err := render.Compile(context.Background(), string(src))
	if err != nil {
		// D2's own compile errors are already well-formatted (file:line:col
		// plus a source snippet) — surfaced verbatim rather than wrapped so
		// none of that context is lost.
		return err
	}

	img, err := render.Render(diagram, *scale)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}
	return nil
}
