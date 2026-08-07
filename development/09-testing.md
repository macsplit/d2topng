# Testing Strategy

The test suite is small and split cleanly by what it's actually verifying:
pure-function unit tests, and full-pipeline golden-image regression tests.

## Unit tests

[`internal/render/units_test.go`](../internal/render/units_test.go) covers
the small, easily-isolated pure functions:

- `TestResolveColor` — every theme token category (`N1`, `B6`, `AA4`) maps
  to the expected `theme` field, and literal colors / `"none"` / `""` pass
  through unchanged. See [Color & Fonts](08-color-and-fonts.md).
- `TestSetColorNoneIsSkipped` — `setColor` returns `false` for `"none"` and
  `""`, `true` for a real token.
- `TestDrawSVGPathRoundTrip` — a closed unit square expressed in D2's
  `M`/`L`/`H`/`V`/`Z` path subset parses and fills without panicking. See
  [`svgpath.go`](06-shapes.md#svgpathgo--replaying-d2s-path-data).
- `TestMidpointHalfwayAlongPolyline` / `TestMidpointSinglePoint` — the
  two-pass length-walk in `midpointAndNormal()` (wrapped by `midpoint()` for
  this test) lands exactly on a route's corner point when that corner is the
  true geometric midpoint, and doesn't panic on a single-point route. See
  [`edges.go`](07-edges.md#midpointandnormal).

[`cmd/d2topng-server/main_test.go`](../cmd/d2topng-server/main_test.go)
covers every HTTP status path of `handleRender` directly via `httptest`,
without a real listener — see [HTTP Server](03-server.md#tests) for the
full list.

## Golden-image tests

[`internal/render/golden_test.go`](../internal/render/golden_test.go) is
the test that actually exercises the full `Compile` → `Render` → `png.Encode`
pipeline end to end, for every `.d2` fixture under
`internal/render/testdata/`, and asserts the encoded PNG bytes are **exactly
identical**, byte for byte, to a checked-in `*.golden.png` of the same name.

This is only possible because the whole pipeline is deterministic: D2's
`d2dagrelayout` and `textmeasure.Ruler` are pure Go with no floating-point
platform variance, and `gg`'s rasterization is pure software (no GPU, no
platform-specific font hinting). That determinism is exactly what makes
byte-exact comparison viable instead of needing perceptual/fuzzy image
diffing — a real advantage of not depending on a browser or system font
renderer for output.

Regenerating goldens after an intentional rendering change:

```
UPDATE_GOLDEN=1 go test ./internal/render/...
```

## A discovered pitfall: stale installed binaries produce silently wrong output

While producing the diagrams in this compendium, several renders of
diagrams containing multi-line **connection labels** (D2 syntax like
`a -> b: "line one\nline two"`) came out as a single line with a `�`
replacement-character glyph in place of the line break, instead of two
separate lines:

> `line one` □ `line two`   *(wrong — one line, tofu box)*

versus the correct:

> `line one`
> `line two`

Multi-line **shape** labels rendered correctly the whole time — only
connection labels were affected. Since both go through the exact same
`drawMultilineAt` function (see
[shapes.go](06-shapes.md#drawlabel-and-drawmultilineat) /
[edges.go](07-edges.md#edge-label)), this looked like it might be a real,
narrow bug specific to the connection-label call path.

**Root cause, after investigation:** it wasn't a code bug at all. The
`d2topng` binary on `$PATH` (`~/go/bin/d2topng`, from a previous
`go install ./cmd/d2topng`) predated the repository's current `HEAD` —
built from an older commit, before whatever change made connection-label
line-splitting correct. A fresh build from the current source
(`go build -o /tmp/d2topng-fresh ./cmd/d2topng`, or simply re-running
`go install ./cmd/d2topng`) rendered the exact same `.d2` input correctly
immediately, with no source changes needed.

**Takeaway for future debugging in this repo:** if rendered output looks
wrong in a way that doesn't match what the current source implies it
should do, check `go install`'s timestamp / rebuild from source before
assuming the code has a bug — `d2topng` is easy to install once and then
silently drift out of date against a repo that keeps moving.

**Fixed:** `cmd/d2topng` now has a `-version` flag (see
[CLI: `-version`](02-cli.md#-version)) that reports either a build-stamped
version (`make build`, via `git describe`) or, for a plain
`go build`/`go install` with no such stamping, the git commit Go embeds
automatically via `runtime/debug.ReadBuildInfo()`. Running
`d2topng -version` next to `git log -1 --format=%h` now answers "is my
installed binary actually current?" directly, without needing to rebuild
and compare output.
