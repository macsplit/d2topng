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
	"runtime/debug"

	"github.com/macsplit/d2topng/internal/render"
)

// version is set at build time via -ldflags "-X main.version=...". Left at
// its zero value for `go install`/`go build` invocations that don't pass
// that flag (e.g. the plain `go install ./cmd/d2topng` from the README) —
// versionString falls back to the VCS revision Go stamps into the binary
// automatically in that case, so `-version` still identifies exactly which
// commit produced the binary rather than reporting nothing.
var version = ""

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "d2topng:", err)
		os.Exit(1)
	}
}

func run() error {
	scale := flag.Float64("scale", 1, "output resolution multiplier (e.g. 2 for double-resolution PNG)")
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: d2topng [-scale N] <input.d2> <output.png>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString())
		return nil
	}

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

// versionString reports a build-stamped version if one was set via
// -ldflags, otherwise the VCS revision Go embeds automatically (unless
// -trimpath was used without -buildvcs=true), otherwise "unknown" — never a
// blank string, so a build with no version information at all is still
// obviously distinguishable from one that successfully resolved a revision.
func versionString() string {
	if version != "" {
		return "d2topng " + version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		var revision string
		dirty := false
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				revision = s.Value
			case "vcs.modified":
				dirty = s.Value == "true"
			}
		}
		if revision != "" {
			if len(revision) > 12 {
				revision = revision[:12]
			}
			if dirty {
				revision += "-dirty"
			}
			return "d2topng dev+" + revision
		}
	}

	return "d2topng unknown"
}
