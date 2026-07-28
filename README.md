# d2topng

Renders [D2](https://d2lang.com) diagrams straight to PNG — no browser, no
SVG step. D2 is used purely as a Go library for parsing, layout, and text
measurement; drawing is done natively with a Go 2D rasterizer
([`gg`](https://github.com/fogleman/gg)) and encoded with the standard
library's `image/png`.

See `PLAN.md` for the design rationale and what's deliberately out of scope
(themes, markdown labels, sketch mode, icons/images).

## Build

```
make build
```

Produces a single static binary at `bin/d2topng` (`CGO_ENABLED=0`, stripped).
No runtime dependencies — fonts are pulled from D2's own embedded font bytes
at build time.

## Install

```
go install ./cmd/d2topng
```

Installs `d2topng` to `$(go env GOPATH)/bin` (usually `~/go/bin`) — add that
to your `PATH` if it isn't already, then run `d2topng` from anywhere.

## Usage

```
bin/d2topng [-scale N] <input.d2> <output.png>
```

- `-scale N` — output resolution multiplier (default 1). Use `-scale 2` or
  higher for crisp high-DPI output; text and strokes stay sharp rather than
  being blurred by upscaling.

## Test

```
make test
```

Includes byte-exact golden-image tests (`internal/render/golden_test.go`)
against fixtures in `internal/render/testdata/`. After an intentional
rendering change, regenerate golden files with:

```
UPDATE_GOLDEN=1 go test ./internal/render/...
```
