# Connection Drawing: `edges.go`

Source: [`internal/render/edges.go`](../internal/render/edges.go) (111
lines).

Everything about drawing one `d2target.Connection`: its route (straight or
curved), arrowheads at either end, and its label.

![Edge drawing](diagrams/07-edge-drawing.png)

## `drawConnection`

```go
func drawConnection(dc *gg.Context, conn d2target.Connection) {
    route := conn.Route
    if len(route) < 2 {
        return
    }
    ...
}
```

A route with fewer than 2 points can't be drawn (no line segment exists) and
is silently skipped — not treated as an error, since D2's layout engine is
trusted to only produce degenerate routes in genuinely degenerate diagrams
(e.g. a self-loop edge case), not as a sign of a bug in this renderer.

### Stroke setup

```go
scaledWidth := float64(conn.StrokeWidth) * outputScale
setColor(dc, conn.Stroke, conn.Opacity)
dc.SetLineWidth(scaledWidth)
if conn.StrokeDash > 0 {
    dashSize, gapSize := svg.GetStrokeDashAttributes(scaledWidth, conn.StrokeDash)
    dc.SetDash(dashSize, gapSize)
} else {
    dc.SetDash()
}
```

Identical pattern to `fillAndStroke` in [shapes.go](06-shapes.md): stroke
width is scaled explicitly by `outputScale` because `gg` applies it in
device pixels, and dash attributes are computed via D2's own
`lib/svg.GetStrokeDashAttributes` so dash/gap sizing matches D2's SVG
output.

### Route: straight segments vs. curves

```go
dc.MoveTo(route[0].X, route[0].Y)
if conn.IsCurve {
    i := 1
    for ; i < len(route)-2; i += 3 {
        dc.CubicTo(route[i].X, route[i].Y, route[i+1].X, route[i+1].Y, route[i+2].X, route[i+2].Y)
    }
} else {
    for i := 1; i < len(route); i++ {
        dc.LineTo(route[i].X, route[i].Y)
    }
}
dc.Stroke()
```

`conn.Route` is a flat `[]*geo.Point`. For curved connections, D2 encodes
each cubic Bézier segment as **3 points** (two control points + an
endpoint) following the starting point, so the loop advances `i += 3` and
draws one `CubicTo` per group. For straight connections, every point after
the first is just a `LineTo`.

### Arrowheads

```go
if conn.DstArrow != d2target.NoArrowhead {
    drawArrowhead(dc, route[len(route)-2], route[len(route)-1], conn.Stroke, conn.Opacity)
}
if conn.SrcArrow != d2target.NoArrowhead {
    drawArrowhead(dc, route[1], route[0], conn.Stroke, conn.Opacity)
}
```

```go
func drawArrowhead(dc *gg.Context, from, tip *geo.Point, strokeToken string, opacity float64) {
    angle := math.Atan2(tip.Y-from.Y, tip.X-from.X)
    const spread = 0.4 // radians
    x1 := tip.X - arrowheadLength*math.Cos(angle-spread)
    y1 := tip.Y - arrowheadLength*math.Sin(angle-spread)
    x2 := tip.X - arrowheadLength*math.Cos(angle+spread)
    y2 := tip.Y - arrowheadLength*math.Sin(angle+spread)

    dc.NewSubPath()
    dc.MoveTo(tip.X, tip.Y)
    dc.LineTo(x1, y1)
    dc.LineTo(x2, y2)
    dc.ClosePath()
    setColor(dc, strokeToken, opacity)
    dc.Fill()
}
```

D2 supports several distinct arrowhead *shapes* (triangle, diamond, circle,
cross, "arrow", ...). This renderer implements exactly **one**: a filled
triangle, `arrowheadLength = 10` units long with a `±0.4` radian spread
around the line's angle at the tip. Every non-`NoArrowhead` value renders
identically as this triangle — which happens to be D2's own default
arrowhead style (`d2target.DefaultArrowhead`), so the common case looks
right, but a diagram explicitly requesting e.g. a diamond or circle
arrowhead will silently get a triangle instead. This is a known, documented
gap rather than a bug.

The source (`from`) and destination (`tip`) points for each arrowhead are
taken from the second-and-last, and first-and-second, points of the route
respectively — i.e. the arrowhead's direction is derived from the route's
own local direction at that end, not from the overall start-to-end vector.

### Edge label

```go
func drawEdgeLabel(dc *gg.Context, conn d2target.Connection, route []*geo.Point) {
    if conn.Label == "" {
        return
    }
    mx, my, nx, ny := midpointAndNormal(route)

    const gap = 4.0
    labelW, labelH := float64(conn.LabelWidth), float64(conn.LabelHeight)
    offset := float64(conn.StrokeWidth)/2 + gap
    lx := mx + nx*(offset+labelW/2)
    ly := my + ny*(offset+labelH/2)

    dc.SetFontFace(fontFace(conn.Bold, float64(conn.FontSize)))
    setColor(dc, conn.GetFontColor(), 1)
    drawMultilineAt(dc, conn.Label, lx, ly)
}
```

Reuses `drawMultilineAt` from [shapes.go](06-shapes.md#drawlabel-and-drawmultilineat)
— the exact same multi-line-safe text drawing routine — but, unlike a shape
label centered on a label-box top-left, positions it at the route's
midpoint **offset perpendicular to the line** by half the label's own size
plus a small gap, so the label sits beside the line rather than on top of
it.

**Why offset instead of drawing on the line:** early on this drew the label
centered directly on the route midpoint, same as D2's own default
(`InsideMiddleCenter`). That put the label's greyish text right on top of
the stroked, bluish line — worst on curved transition arrows, where the
stroke crosses the text diagonally rather than just skimming underneath a
couple of glyphs. D2's own SVG renderer tolerates on-line labels because it
papers over the problem with an SVG `<mask>` that cuts a hole in the
connection's stroke behind the label (`d2svg.makeLabelMask`); a first fix
here imitated that with a plain opaque rect in the page background color
before drawing the text (no real SVG masking available in a raster
renderer). That worked for the common case but had a real flaw: unlike an
SVG mask, which only ever removes stroke, an opaque rect **paints over**
whatever else happens to be underneath — so a label landing on top of
another shape's differently-colored fill, not just bare canvas, would look
wrong. Spacing the label off the line entirely avoids that failure mode, at
the cost of no longer matching D2's exact on-line placement.

`nx, ny` is the unit vector perpendicular to the route segment the midpoint
falls on, always oriented "up" (toward decreasing Y — see `segmentNormal`
below) so a label lands on a consistent side of the line rather than
flipping above/below depending on which way a given edge happens to be
routed. `offset` clears the line's own stroke width plus a fixed 4px gap;
multiplying by `nx`/`ny` separately for the width and height halves (rather
than a single scalar offset) mirrors the same approximation D2's own
`label.getOffsetLabelPosition` uses for outside labels — exact for
axis-aligned segments, an approximation for diagonal ones, which is fine
since `d2dagrelayout`'s routes are almost always axis-aligned apart from
short corner curves.

**Known limitation:** because the offset math assumes a roughly
axis-aligned segment, a label on a steeply diagonal segment (most likely
the short curved corner of a route, rather than a long midsection) can end
up offset less cleanly than on a horizontal or vertical one. This wasn't
visible in local testing but is a known approximation, not a proven-exact
guarantee, for that case.

### `midpointAndNormal`

```go
func midpointAndNormal(route []*geo.Point) (x, y, nx, ny float64) {
    total := 0.0
    for i := 1; i < len(route); i++ {
        total += geo.EuclideanDistance(route[i-1].X, route[i-1].Y, route[i].X, route[i].Y)
    }

    target := total / 2
    walked := 0.0
    for i := 1; i < len(route); i++ {
        a, b := route[i-1], route[i]
        seg := geo.EuclideanDistance(a.X, a.Y, b.X, b.Y)
        if walked+seg >= target {
            t := 0.0
            if seg > 0 {
                t = (target - walked) / seg
            }
            nx, ny := segmentNormal(a, b)
            return a.X + t*(b.X-a.X), a.Y + t*(b.Y-a.Y), nx, ny
        }
        walked += seg
    }
    last := route[len(route)-1]
    nx, ny = 0, -1
    if len(route) >= 2 {
        nx, ny = segmentNormal(route[len(route)-2], last)
    }
    return last.X, last.Y, nx, ny
}
```

Two-pass algorithm: first computes the route's **total Euclidean length**
by summing every consecutive-point distance (treating the route as a
polyline — this does not account for `IsCurve` routes actually being
smooth Béziers, so the midpoint of a curved edge's label position is an
approximation based on its control polygon's length, not the true arc
length of the curve). Second pass walks the same segments accumulating
distance until it finds the segment containing the halfway point, then
linearly interpolates (`t`) within that segment to the exact point, and
resolves the perpendicular normal (`segmentNormal`, below) for that same
segment.

The final `last.X, last.Y` branch is a defensive fallback for a
single-point or otherwise degenerate route where the walk never finds a
segment satisfying `walked+seg >= target` — it can't actually be reached
for `len(route) >= 2` given the two-pass logic, but keeps the function
total and panic-free for any input; the `len(route) >= 2` guard around the
`segmentNormal` call there specifically covers the single-point case, where
there is no `route[len(route)-2]` to take a normal against.

Notably, `midpointAndNormal` (via the `midpoint` wrapper, kept for the unit
test below) is only ever called from `drawEdgeLabel`, and `drawConnection`
already returns early for `len(route) < 2`, so in practice it's never
invoked with fewer than 2 points — but see [Testing Strategy](09-testing.md)
for the unit test that exercises the single-point case directly regardless.

### `segmentNormal`

```go
func segmentNormal(a, b *geo.Point) (nx, ny float64) {
    if a.X == b.X && a.Y == b.Y {
        return 0, -1
    }
    nx, ny = geo.GetUnitNormalVector(a.X, a.Y, b.X, b.Y)
    if ny > 0 {
        nx, ny = -nx, -ny
    }
    return nx, ny
}
```

Wraps D2's own `geo.GetUnitNormalVector` (used the same way by
`label.getOffsetLabelPosition` for outside shape/border labels) with two
adjustments: a zero-length-segment guard (`a == b`, which would otherwise
divide by zero inside `GetUnitNormalVector` and propagate `NaN`/`Inf` all
the way to the drawn label position), and flipping the vector so `ny` is
always `<= 0` — i.e. always pointing "up" in image space, since D2's Y axis
increases downward — so labels consistently land above their line rather
than alternating sides depending on the segment's direction of travel.
