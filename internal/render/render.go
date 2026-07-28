// Package render turns a compiled D2 diagram into a PNG using a native Go
// rasterizer (gg), with no SVG/HTML/browser step involved.
package render

import (
	"context"
	"fmt"
	"image"
	"sort"

	"github.com/fogleman/gg"
	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2target"
	d2log "oss.terrastruct.com/d2/lib/log"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

// pad matches d2svg's DEFAULT_PADDING so layouts feel the same as `d2 render`.
const pad = 100

// outputScale is a supersampling factor applied to the whole render. gg's
// transform matrix (set via dc.Scale below) automatically scales all shape/
// path geometry, but NOT stroke width, dash lengths, or font size — those are
// applied in device pixels rather than transformed path coordinates, so
// fonts.go and the stroke/dash call sites in shapes.go/edges.go read this to
// scale themselves explicitly. Fine as a package-level var since Render is
// not called concurrently within a single CLI invocation.
var outputScale = 1.0

// Compile parses and lays out D2 source, returning the renderer-agnostic
// scene graph we draw from. Layout is always d2dagrelayout (pure Go, no
// JVM/WASM) — there is no other layout engine wired in.
func Compile(ctx context.Context, dsl string) (*d2target.Diagram, error) {
	ctx = d2log.WithDefault(ctx)

	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, fmt.Errorf("init text ruler: %w", err)
	}

	diagram, _, err := d2lib.Compile(ctx, dsl, &d2lib.CompileOptions{
		Ruler: ruler,
		LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
			return d2dagrelayout.DefaultLayout, nil
		},
	}, nil)
	if err != nil {
		return nil, err
	}
	return diagram, nil
}

// Render draws diagram's shapes and labels onto an in-memory RGBA image.
// scale is a supersampling factor (1 = D2's native coordinate resolution,
// 2 = double-resolution output, etc); values <= 0 are treated as 1.
func Render(diagram *d2target.Diagram, scale float64) (image.Image, error) {
	if scale <= 0 {
		scale = 1
	}
	outputScale = scale

	tl, br := diagram.BoundingBox()
	width := br.X - tl.X + pad*2
	height := br.Y - tl.Y + pad*2

	dc := gg.NewContext(int(float64(width)*scale), int(float64(height)*scale))
	dc.SetHexColor(theme.Neutrals.N7)
	dc.Clear()
	dc.Scale(scale, scale)

	// Shift every draw so the bounding box's top-left lands at (pad, pad).
	dc.Translate(float64(pad-tl.X), float64(pad-tl.Y))

	// Shapes and connections are drawn together in ZIndex order, matching
	// how D2 itself layers them (see d2svg's zIndex sort) rather than always
	// drawing all shapes then all edges.
	type drawable struct {
		zIndex int
		draw   func()
	}
	var items []drawable
	for _, shape := range diagram.Shapes {
		shape := shape
		items = append(items, drawable{shape.ZIndex, func() { drawShape(dc, shape) }})
	}
	for _, conn := range diagram.Connections {
		conn := conn
		items = append(items, drawable{conn.ZIndex, func() { drawConnection(dc, conn) }})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].zIndex < items[j].zIndex })
	for _, item := range items {
		item.draw()
	}

	return dc.Image(), nil
}
