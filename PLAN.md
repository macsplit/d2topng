# d2topng — Implementation Plan

Native, browser-free D2 → PNG renderer in Go.

## Key insight that changes the spec's scope

The spec assumes we need to hook D2 halfway through compilation and hand-roll
scene-graph extraction and text measurement. We don't. D2 already exposes a
**renderer-agnostic scene graph**: `d2target.Diagram`, produced by
`d2lib.Compile()`. Today D2's only consumer of that struct is `d2svg`, which
turns it into SVG/XML. Our job is to write a **second consumer** —
`d2png` — that turns the same struct into pixels via a Go 2D rasterizer,
instead of writing markup that a browser would later rasterize.

That also means text measurement is already solved: D2's
`lib/textmeasure` package measures glyphs natively (via `golang.org/x/image/font`
+ `sfnt`, using the same font files it ships for SVG output) and runs during
`d2lib.Compile()`, before we ever see the diagram. There is no browser or
chromedp step in modern D2 — the "web way" described in the spec was already
replaced upstream. So Stage 1 and Stage 2 of the pipeline are "call D2's Go
API"; the real engineering is Stage 3 (raster backend) and Stage 4 (encode).

Revised pipeline:

```
D2 DSL text
   │  d2lib.Compile(...)   [reuse D2 as a library — parsing, layout, text measurement]
   ▼
*d2target.Diagram          (shapes, connections, labels, absolute coords, colors, theme)
   │  d2png.Render(diagram)  [NEW — the only code we actually write]
   ▼
*image.RGBA
   │  image/png.Encode
   ▼
out.png
```

## Decisions locked in

- **Lightest-weight solution wins.** No configurability for its own sake.
- **Layout engine: `d2dagrelayout` only.** Pure Go, no JVM/WASM runtime, no
  multi-MB embedded blob. `d2elklayout` is not wired in at all — not even as
  an opt-in flag — to keep the dependency tree and binary size minimal.
- **Themes: out of scope entirely.** No `d2themes` catalog, no `--theme` flag,
  no per-theme palette table. One opinionated, hardcoded palette (colors,
  default stroke widths, corner radii, font) baked into `internal/render`.
  If a `.d2` file sets explicit colors/styles inline, those are honored (they
  come through on `d2target.Shape`/`Connection` fields regardless of theme
  machinery); only the theme *selection* system is dropped.
- **Sketch mode, markdown labels, icons/images: dropped, not deferred.** These
  exist in the spec's trade-off table precisely because they require
  reimplementing browser behavior — the opposite of lightweight. Plain text
  labels only (single font, single weight, manual line-wrap on explicit
  newlines). Revisit only if explicitly requested later.
- **Target D2 version: `oss.terrastruct.com/d2 v0.7.1`** (latest stable at time
  of writing). Note the import path is `oss.terrastruct.com/d2`, not
  `github.com/terrastruct/d2`.

## Phase 0/1 spike — confirmed findings (2026-07-29)

A working spike (`cmd/spike`) compiled a 3-node/2-edge `.d2` string via
`d2lib.Compile` with a `LayoutResolver` wired to `d2dagrelayout.DefaultLayout`
and a `textmeasure.NewRuler()` passed as `CompileOptions.Ruler`, then dumped
the resulting `*d2target.Diagram` as JSON. Findings that change downstream
work:

- **Toolchain:** this machine only has Go 1.22 installed, but `GOTOOLCHAIN=auto`
  (the default) transparently downloads Go 1.25.12 to satisfy D2's
  `go >= 1.24` requirement. No manual Go upgrade needed; `go.mod` pins
  `go 1.24.0` / `toolchain go1.25.12`.
- **No Playwright in our binary.** Verified with `go tool nm` on the built
  spike binary — zero `playwright` symbols. It's a dependency of D2's own
  browser-based PNG/PDF export path (`d2exporter`/CLI), which we never import
  a code path that reaches. The "lightweight, no browser" premise holds.
- **Text measurement is genuinely native**, no browser: `textmeasure.NewRuler()`
  loads embedded fonts and measures glyphs directly. This must be constructed
  and passed as `CompileOptions.Ruler` — omitting it causes a hard runtime
  error ("dimensions for object label ... not found"), it is not optional.
- **Correction: shape/connection colors are theme tokens, not hex.**
  `d2target.Shape.Fill` / `.Stroke` / `.Color` come back as strings like
  `"B6"`, `"B1"`, `"N1"`, `"N2"`, `"AA4"` — palette-slot codes from
  `d2themescatalog`, resolved to real colors only inside `d2svg`'s rendering
  step (which we're bypassing). So `palette.go` isn't a nice-to-have — it must
  contain a hardcoded token→hex table for exactly D2's default theme (theme
  ID 0). Get this table directly from `d2themescatalog`'s default theme
  source in the pinned version, not by guessing swatches.
- **Layout engine detail:** `d2dagrelayout.DefaultLayout` runs the real
  `dagre.js` through `goja` (a pure-Go embedded JS interpreter), not
  hand-written Go graph-layout math. Still zero browser/JVM/WASM — consistent
  with "lightweight" — just note it's JS-via-goja under the hood when
  debugging layout quirks.
- API shape to reuse going forward:
  ```go
  ruler, _ := textmeasure.NewRuler()
  diagram, _, err := d2lib.Compile(ctx, dsl, &d2lib.CompileOptions{
      Ruler: ruler,
      LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
          return d2dagrelayout.DefaultLayout, nil
      },
  }, nil)
  ```
  `cmd/spike/main.go` has the full working example; fold this into
  `internal/render`'s compile step rather than duplicating it, and delete
  `cmd/spike` once `cmd/d2topng` supersedes it.

## Repo / module layout

```
d2topng/
  go.mod
  cmd/d2topng/main.go          # CLI entrypoint (flag parsing, file I/O)
  internal/render/
    render.go                  # Diagram -> *gg.Context orchestration
    shapes.go                  # per-d2target.Shape drawing (rect, oval, diamond,
                                #   cylinder, cloud, person, package, class/sql
                                #   table shapes, etc.)
    edges.go                   # connections: bezier/polyline paths, arrowheads,
                                #   edge labels
    text.go                    # label placement, single-font plain-text
                                #   multi-line wrap (no markdown)
    palette.go                 # single hardcoded default palette (no theme
                                #   selection); explicit inline .d2 colors
                                #   still pass through untouched
    fonts.go                   # go:embed one default TTF (reuse D2's bundled
                                #   font if license permits, else Go's own)
  internal/render/testdata/     # golden PNGs + fixture .d2 files
  internal/render/golden_test.go
  assets/fonts/                 # embedded font files
  README.md
```

## Phased delivery

### Phase 0 — Project setup
- `go mod init`, pin Go version, add `github.com/terrastruct/d2` and
  `github.com/fogleman/gg` as dependencies.
- Confirm D2's license (MPL-2.0) is compatible with intended distribution;
  keep NOTICE/license file.
- Commit skeleton to git (already initialized) as first commit.

### Phase 1 — Spike: get a Diagram out
- Smallest possible program: `d2lib.Compile(ctx, dsl, &d2lib.CompileOptions{})`
  → dump the returned `*d2target.Diagram` as JSON.
- Goal: confirm exact field names/types for shapes, connections, and how
  fonts/colors are represented in this D2 version. This determines the real
  shape of `shapes.go`/`edges.go` — don't hand-design those before this spike.

### Phase 2 — Minimal raster renderer — DONE (2026-07-29)
- Implemented in `internal/render/{render,palette,fonts}.go` +
  `cmd/d2topng/main.go`. Builds a `gg.Context` sized from
  `diagram.BoundingBox()` (+100px pad, matching `d2svg.DEFAULT_PADDING`), fills
  background with the theme's `N7`, draws each shape as a plain/rounded
  rectangle with fill+stroke resolved via `palette.go`'s token table, and
  centers the label using D2's own embedded SourceSansPro TTF bytes
  (`d2fonts.FontFaces`) parsed with `golang/freetype/truetype`.
- Non-rectangle shape types (e.g. `cylinder`) currently draw as plain
  rectangles — intentional, deferred to Phase 4. Edges are not drawn yet —
  Phase 3.
- One fix needed beyond the original plan: D2 logs a noisy WARN
  ("missing slog.Logger in context") unless the context passed to
  `d2lib.Compile` is wrapped with `d2log.WithDefault(ctx)` first
  (`oss.terrastruct.com/d2/lib/log`). Now wired into `render.Compile`.
- Verified visually: `x -> y -> z` with `z.shape: cylinder` renders three
  boxes with correct theme colors (blue fill/stroke) and centered bold
  labels.
- `cmd/spike` removed now that `cmd/d2topng` supersedes it.

### Phase 3 — Edges — DONE (2026-07-29)
- Implemented in `internal/render/edges.go`, wired into `render.go` via a
  combined shapes+connections draw list sorted by `ZIndex` (matching how
  `d2svg` layers them, rather than always drawing all shapes before all
  edges).
- Path construction: `IsCurve` connections replay D2's own cubic-bezier
  route encoding (groups of 3 route points per `C` segment, same convention
  as `d2svg.pathData`); non-curve connections currently draw as a straight
  polyline through the route points, without `d2svg`'s corner-radius arc
  smoothing — an intentional simplification for v1 (sensible default, not
  pixel-parity).
- Arrowheads: every non-`none` `Arrowhead` value renders as a filled
  triangle. D2 has several arrowhead shapes (`arrow`, `unfilled-triangle`,
  `cf-many-required`, etc.) but `TriangleArrowhead` is D2's own
  `DefaultArrowhead`, so this covers the overwhelming common case; the rest
  is deferred (would be a Phase 5 styling-fidelity item if ever needed).
- Edge labels: placed at the geometric midpoint by cumulative path length
  (not `LabelPosition`/`LabelPercentage` yet), horizontally/vertically
  centered on that point.
- Verified visually on both a straight chain (`x -> y -> z`) and a
  branch/merge diamond (`a->b, a->c, b->d, c->d`) — curves, arrowhead
  orientation, and the "merge" edge label all render correctly.

### Phase 4 — Full shape catalog — mostly DONE (2026-07-29)
- **Key reuse discovery**: `oss.terrastruct.com/d2/lib/shape` already exposes
  each shape's outline as `Shape.GetSVGPathData() []string` — an SVG path
  "d" string using only the M/L/H/V/C/Z subset (no elliptical arcs). Rather
  than hand-coding geometry per shape type, `internal/render/svgpath.go`
  implements a small generic parser for exactly that subset and replays it as
  `gg` draw calls. `d2target.DSL_SHAPE_TO_SHAPE_TYPE` (also public) maps a
  shape's DSL type string (`"cylinder"`, `"diamond"`, ...) to the
  `lib/shape` type constant, so `drawPathShape` in `shapes.go` is one
  generic function covering cylinder, diamond, cloud, person, hexagon, page,
  document, parallelogram, queue, package, step, callout, stored_data — no
  per-shape Go code needed.
- **Oval/circle special case**: `lib/shape`'s oval and circle types return no
  path data at all — D2's own `d2svg` draws them as a native SVG `<ellipse>`
  instead (`renderOval`, `d2svg.go:1346`). Confirmed by comparing against the
  actual `d2` CLI binary installed on this machine (`d2 test3.d2 ref.svg`):
  `shape: circle` compiles down to `type: "oval"` with width==height, not a
  literal circle type — so oval and circle are the same code path. Added
  `drawEllipseShape` (`dc.DrawEllipse`, centered, rx=w/2, ry=h/2) as its own
  branch in `shapes.go`, checked before falling through to `drawPathShape`.
  Lesson: when unsure whether an edge case matches D2's real output, compare
  against the real `d2` binary rather than assuming from `lib/shape` alone.
- **Label positioning fixed generically, not per-shape**: labels were
  initially always centered in the shape's bounding box, which looked wrong
  for shapes like `person` (D2 places that label outside/below by default).
  Rather than special-casing person, `drawLabel` now calls
  `label.FromString(s.LabelPosition).GetPointOnBox(box, label.PADDING,
  labelW, labelH)` — another direct reuse of D2's own positioning logic
  (`lib/label`), which already returns the correct top-left point for all
  13 INSIDE/OUTSIDE/BORDER position variants D2 can assign per shape type.
- **Still composite/deferred** (`compositeShapeTypes` in `shapes.go`, drawn as
  plain rectangles for now): `class` and `sql_table` (D2 renders these as
  bespoke multi-row tables via `drawClass`/`drawTable` in `d2svg.go`, not a
  single path) and `image` (would need icon fetch/embed policy, out of scope
  per the earlier "no icons" decision). Sequence diagrams are structurally a
  different layout mode entirely (lifelines/lanes), not attempted.
- Verified visually against a diagram exercising cylinder/diamond/person/
  cloud/hexagon/oval together — all shapes, fills, strokes, and (for person)
  label placement match expected D2 output.

### Phase 5 — Styling fidelity — mostly DONE (2026-07-29)
- **Opacity**: `internal/render/color.go`'s `setColor` parses any D2 color
  field (theme token, hex, or CSS name) via the already-transitive
  `github.com/mazznoer/csscolorparser` dependency (pulled in by D2 itself, so
  no new dependency weight) and multiplies alpha by the shape/connection's
  `Opacity`. Returns `false` for `""`/`"none"` (D2's sentinel for "don't draw
  this") so callers can skip fill/stroke entirely — needed once opacity-aware
  drawing made "just always fill" wrong.
- **Stroke dash**: reused `oss.terrastruct.com/d2/lib/svg.GetStrokeDashAttributes`
  (the same function `d2target.Shape.CSSStyle()`/`Connection.CSSStyle()` use)
  to convert `StrokeDash` into `gg.SetDash(dashSize, gapSize)` — same formula
  D2 itself uses, not a guessed dash ratio.
- **Shadow, "multiple", double-border**: `shapes.go` refactored around one
  `drawOutline(dc, s, x, y, w, h, fill, stroke, ...)` function that traces +
  fills/strokes a shape at an arbitrary box, reused for the shape's main body
  *and* its decorations by just calling it again at an offset/inset box:
  shadow = offset copy in dark translucent fill only; multiple = full offset
  copy behind (`d2target.MULTIPLE_OFFSET`); double-border = inset outline-only
  copy (`d2target.INNER_BORDER_OFFSET`). This makes all three decorations work
  uniformly across every shape type (rect, oval, path-based), which is
  actually broader than D2's own SVG renderer (which only wires
  multiple/double-border up for rectangle and oval) — an intentional
  simplification in the "lightweight, opinionated" direction rather than
  matching D2's exact per-shape-type feature matrix.
- **3D**: `draw3D` in `shapes.go` approximates D2's isometric rectangle look
  (back copy offset by `d2target.THREE_DEE_OFFSET`, two connecting
  quadrilateral panels, front face on top) — only wired up for rect-like
  shapes, matching D2's own scope (`d2svg` only implements 3D for the
  rectangle case).
- **Verified against ground truth**: this machine has the real `d2` v0.7.1
  CLI installed, so rather than eyeballing our own output in isolation, we
  generated `d2 test.d2 ref.svg` and rasterized it with `inkscape
  --export-type=png` for a side-by-side comparison. Dashed, shadow, 3D,
  multiple, and opacity/fade all matched closely. (One apparent mismatch —
  a solid black double-border box in the Inkscape render — was confirmed to
  be an Inkscape SVG font-face parsing artifact from the reference file
  itself, not a real rendering difference; our double-line geometry matched
  the reference's border structure.) Worth remembering for later phases:
  **compare against the real `d2` binary + a headless SVG rasterizer
  (inkscape worked; chromium-browser's snap sandboxing did not) instead of
  just visually inspecting our own PNG.**
- **Deferred**: `FillPattern` (dot/paper texture overlays) — cosmetic texture
  layered on top of a fill, no correctness value, dropped per the
  "lightweight" directive rather than reimplementing D2's pattern rasterizer.

### Phase 6 — CLI polish — DONE (2026-07-29)
- `d2topng [-scale N] <input.d2> <output.png>`. No `-layout`, `-theme`, or
  `-sketch` flags — those axes don't exist in this build.
- Compile errors from `d2lib.Compile` are returned to the CLI unwrapped —
  D2's own parser errors already come formatted as `file:line:col: message`
  with a source snippet, so wrapping them with extra `fmt.Errorf` context
  would just add noise.
- **`-scale` surfaced a real gg gotcha worth remembering**: gg's transform
  matrix (`dc.Scale`) automatically scales path/shape geometry (confirmed by
  reading `MoveTo`/`LineTo`/`CubicTo` — they all call `TransformPoint`), but
  it does **not** scale stroke width, dash lengths, or font rendering size —
  those are applied directly in device pixels. Worse, `gg`'s word-wrap
  (`DrawStringWrapped`) measures the wrap width against the font face
  directly, ignoring the matrix entirely — so naively scaling only the font
  size (to keep text crisp at higher output resolution) while leaving
  position/width flowing through the matrix caused wrapping to trigger as if
  the available width had shrunk by the scale factor.
  Fix, in `internal/render`: `outputScale` (package var, set once per
  `Render` call) is applied explicitly at every place gg doesn't scale
  automatically — `fonts.go`'s `fontFace` multiplies point size by it;
  `shapes.go`/`edges.go` multiply `StrokeWidth` before `SetLineWidth`/dash
  calculation. For text specifically, position and wrap width are resolved to
  already-transformed device-pixel coordinates via `dc.TransformPoint`, then
  drawn under a temporary identity matrix (`dc.Push()`/`dc.Identity()`/...
  /`dc.Pop()`) with the wrap width pre-multiplied by `outputScale` — this
  keeps position, wrap width, and font size all in the same unit space.
  Verified at `-scale 3`: output is exactly 3x the base pixel dimensions,
  text is crisp (not blurred/upscaled) and wraps identically to `-scale 1`.

### Phase 7 — Testing — DONE (2026-07-29)
- Golden-image tests (`internal/render/golden_test.go` +
  `internal/render/testdata/*.d2`): compiles+renders each fixture and
  compares the encoded PNG **byte-for-byte** against a checked-in
  `*.golden.png`, not a perceptual/tolerance diff. This works because the
  whole pipeline is deterministic pure Go — D2's layout/text-measurement and
  our own software rasterizer (`gg`, no OS font hinting or GPU involved) — no
  antialiasing variance across machines to buffer against. Confirmed by
  running the suite 3x in a row (`go test -count=3`) with identical output
  each time before trusting exact-match comparison over a tolerance-based
  one. Regenerate with `UPDATE_GOLDEN=1 go test ./internal/render/...` after
  an intentional rendering change.
- Three fixtures cover the ground gained in Phases 2-5: `simple.d2` (chain +
  cylinder + edge label), `shapes.d2` (cylinder/diamond/person/cloud/hexagon/
  oval-as-circle together), `styles.d2` (dash/shadow/3d/multiple/
  double-border/opacity together).
- Unit tests (`internal/render/units_test.go`) for the pure logic that's easy
  to get subtly wrong but hard to eyeball in a rendered PNG: `resolveColor`
  token mapping, `setColor`'s `"none"`/`""` skip behavior, the SVG path-subset
  parser (`drawSVGPath`) round-tripping M/L/H/V/Z, and `midpoint`'s
  cumulative-length walk (including the single-point edge case).
- `go test ./...` passes; `gofmt -l .` and `go vet ./...` clean.

### Phase 8 — Packaging — DONE (2026-07-29)
- `make build` → `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"` produces
  `bin/d2topng`: confirmed statically linked (`file`/`ldd` — "not a dynamic
  executable") at ~24MB, no CGO. No separate font file to embed — fonts come
  from D2's own `d2fonts.FontFaces` bytes at build time (see Phase 2), so
  there's nothing extra of ours to `go:embed`.
- Re-confirmed on the actual release-built binary (not just `go run`/`go
  build` during development) that Playwright never links in
  (`go tool nm bin/d2topng | grep -i playwright` — empty).
- Added `Makefile` (`build`/`test`/`fmt`/`vet`/`clean`), `.gitignore` for
  `/bin/`, and a short `README.md` covering build/usage/test — GoReleaser /
  cross-platform build automation intentionally skipped as unnecessary
  process weight for a single-binary Go CLI at this stage; revisit only if
  multi-platform release distribution is actually requested.

## Status: all 8 phases complete (2026-07-29)
Core pipeline (D2 parse/layout/measure → native scene graph → `gg` raster →
PNG), full shape catalog, styling fidelity, `-scale` output, tests, and a
static binary are all in place. Nothing currently planned is outstanding;
remaining ideas (markdown labels, sketch mode, icons, class/sql_table
composite shapes) are explicit non-goals unless raised again later.

## Phase 9 — HTTP service (added after v1) — DONE (2026-07-29)
- Decided against MCP for remote/hosted use: MCP's remote transport
  (Streamable HTTP) is newer/less uniformly supported across SDKs, whereas a
  plain `POST /render` HTTP endpoint needs no protocol-specific client
  support at all — any agent that can issue an HTTP request can use it.
- **Same repo, not a sibling one** — the deciding factor was Go's `internal/`
  visibility rule: `internal/render` can only be imported by code inside this
  module, so a separate repo couldn't import it without either exporting the
  package (dropping the boundary that's intentionally keeping it a non-public
  API for now) or cross-repo version pinning. `cmd/d2topng-server` sits
  alongside `cmd/d2topng`, both importing `internal/render` directly.
- `cmd/d2topng-server/main.go`: `GET /healthz` (liveness, no auth) and
  `POST /render[?scale=N]` (raw D2 source body → `image/png`, or `400` with
  D2's own compile diagnostics as the body). Optional bearer-token auth via
  `D2TOPNG_API_TOKEN` — open if unset (fine for local use, not for a public
  deploy). 1MiB request body cap via `http.MaxBytesReader`, 20s compile+render
  timeout via `context.WithTimeout`. Verified end to end with `curl`:
  healthz, missing/wrong token → 401, valid render → 200 + real PNG, invalid
  D2 → 400 with diagnostics, `?scale=2` → exactly double resolution.
- Tests in `cmd/d2topng-server/main_test.go` via `httptest`, no real network.
- **Render.com deploy researched before implementing, not assumed**: fetched
  Render's current docs (native Go runtime docs, Blueprint spec, Go
  net/http quickstart) rather than relying on possibly-stale prior knowledge.
  Key facts that shaped `render.yaml`: Render's native Go runtime is
  currently pinned to Go 1.24, which satisfies this repo's `go 1.24.0` +
  `toolchain go1.25.12` — `GOTOOLCHAIN=auto` will fetch 1.25.12 during
  Render's build step same as it does locally, since build environments need
  outbound network for module fetching regardless. Services must bind to
  Render's `$PORT` env var (default 10000) — implemented in `main.go`.
  Render's zero-config path is specifically a **Blueprint** (`render.yaml` at
  repo root) connected via GitHub/GitLab — arbitrary git remotes (e.g. the
  `nuc` bare repo this project also pushes to) don't support the Blueprint
  auto-deploy flow, only GitHub/GitLab do.
- `render.yaml`: `type: web`, `runtime: go`, `healthCheckPath: /healthz`,
  `D2TOPNG_API_TOKEN` declared with `sync: false` (Render's convention for a
  secret set manually in the dashboard rather than committed to git).

## Explicit non-goals (not deferred — dropped)
- Themes / theme selection.
- Markdown / rich text in labels.
- Sketch mode.
- Icons/images (remote fetch), animation, tooltips, links — PNG is static and
  the goal is a small dependency footprint, not feature parity with `d2 render`.

## Risks
- `d2target.Diagram`'s internal shape is not a stable public API guarantee —
  pin an exact D2 version in `go.mod` and re-run the Phase 1 spike on any
  upgrade before touching downstream code.
- Shape/theme visual fidelity vs. official `d2 render` output can only be
  verified by side-by-side comparison, not assumption — do this continuously,
  not just at the end.
