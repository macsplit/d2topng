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
- `-version` — print version info and exit. `make build` stamps a real
  version (from `git describe`); a plain `go build`/`go install` falls back
  to the git commit Go embeds automatically, so `d2topng -version` always
  tells you exactly which commit produced the binary you're running — handy
  for confirming an installed binary isn't stale after pulling new changes.

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

## HTTP server

`cmd/d2topng-server` exposes the same renderer as a plain HTTP service —
`POST` D2 source, get a PNG back — instead of MCP, so any agent or script
that can issue an HTTP request can use it without protocol-specific client
support.

```
make build-server
PORT=8080 D2TOPNG_API_TOKEN=some-secret bin/d2topng-server
```

- `GET /healthz` — liveness check, no auth required.
- `POST /render[?scale=N]` — request body is raw D2 source; response is
  `image/png`. Compile errors come back as `400` with D2's own diagnostics
  as the body. `scale` is read only from this query parameter — there is no
  equivalent request header, so a `Scale:` header (or similar) is silently
  ignored.
- If `D2TOPNG_API_TOKEN` is set, requests must include
  `Authorization: Bearer <token>`. If unset, the endpoint is open — fine for
  local use, not recommended for a public deployment.

```
curl -H "Authorization: Bearer some-secret" \
  --data-binary @diagram.d2 \
  http://localhost:8080/render -o diagram.png
```

### Deploying to Render.com

`render.yaml` at the repo root is a Blueprint: connect this GitHub repo via
Render's "New → Blueprint" flow and it builds/deploys with no further
configuration beyond setting the `D2TOPNG_API_TOKEN` secret in the Render
dashboard (it's declared `sync: false` in the blueprint, i.e. not stored in
git). The service binds to Render's `$PORT` automatically.

## License

MIT — see `LICENSE`. D2 itself (`oss.terrastruct.com/d2`) is MPL-2.0, used
here unmodified as a library dependency (MPL-2.0's copyleft applies at the
file level to changes to D2's own source, not to code that merely imports
it), so it doesn't affect this repo's license. Other dependencies
(`fogleman/gg`, `mazznoer/csscolorparser`) are MIT.
