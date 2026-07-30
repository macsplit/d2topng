# d2topng — Development Compendium

This is a from-the-source-code walkthrough of how `d2topng` actually works
internally: every Go file, every function, and how they connect. It's aimed
at anyone changing the renderer, not at end users (see the repo root
[`README.md`](../README.md) for usage).

The whole codebase is small — about 1,000 lines of Go across two commands
and one internal package — so this compendium tracks it 1:1 rather than
grouping loosely-related code together.

Every diagram in this compendium is a `.d2` source file rendered to PNG with
the project's own renderer (`d2topng` itself — see
[`diagrams/`](diagrams/) for the source). There is no external diagramming
tool involved; the tool documents itself with itself.

## Contents

1. [Architecture Overview](01-architecture.md) — the whole system in one picture: entrypoints, the `internal/render` package, and how they use `oss.terrastruct.com/d2` as a library.
2. [CLI: `cmd/d2topng`](02-cli.md) — the single-shot command-line tool: flag parsing, file I/O, error surfacing.
3. [HTTP Server: `cmd/d2topng-server`](03-server.md) — the `POST /render` service: auth, request limits, timeouts, error responses.
4. [Compile Pipeline: `render.Compile`](04-compile.md) — turning D2 source text into a `*d2target.Diagram` scene graph via the D2 library.
5. [Render Pipeline: `render.Render`](05-render-pipeline.md) — turning a scene graph into an `image.Image`: bounding box, scale, coordinate transforms, z-ordering.
6. [Shape Drawing: `shapes.go`](06-shapes.md) — how each D2 shape type (rectangle, oval, diamond, 3D, multiple, double-border, shadow...) gets rasterized.
7. [Connection Drawing: `edges.go`](07-edges.md) — routes, curves, arrowheads, and edge labels.
8. [Color & Fonts: `color.go`, `palette.go`, `fonts.go`](08-color-and-fonts.md) — resolving D2 theme tokens to concrete colors, and loading D2's embedded font as scaled `font.Face`s.
9. [Testing Strategy](09-testing.md) — golden-image tests, unit tests, and the one real bug this compendium's own diagrams turned up while being produced.

## How the diagrams were produced

Each `.d2` file in [`diagrams/`](diagrams/) was rendered with:

```
d2topng -scale 2 diagrams/NN-name.d2 diagrams/NN-name.png
```

`-scale 2` was used throughout for crisper text at the pixel dimensions
these diagrams end up at. `d2topng` was built from this repository's own
`HEAD` (`go install ./cmd/d2topng`) before rendering — see
[Testing Strategy](09-testing.md) for why that mattered.
