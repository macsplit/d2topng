package render

import (
	"math"

	"github.com/fogleman/gg"
	"oss.terrastruct.com/d2/d2target"
	"oss.terrastruct.com/d2/lib/geo"
	"oss.terrastruct.com/d2/lib/svg"
)

const arrowheadLength = 10.0

func drawConnection(dc *gg.Context, conn d2target.Connection) {
	route := conn.Route
	if len(route) < 2 {
		return
	}

	// Stroke width and dash lengths are applied in device pixels by gg
	// (unlike route geometry, which the transform matrix scales
	// automatically), so outputScale must be applied here explicitly.
	scaledWidth := float64(conn.StrokeWidth) * outputScale
	setColor(dc, conn.Stroke, conn.Opacity)
	dc.SetLineWidth(scaledWidth)
	if conn.StrokeDash > 0 {
		dashSize, gapSize := svg.GetStrokeDashAttributes(scaledWidth, conn.StrokeDash)
		dc.SetDash(dashSize, gapSize)
	} else {
		dc.SetDash()
	}

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

	if conn.DstArrow != d2target.NoArrowhead {
		drawArrowhead(dc, route[len(route)-2], route[len(route)-1], conn.Stroke, conn.Opacity)
	}
	if conn.SrcArrow != d2target.NoArrowhead {
		drawArrowhead(dc, route[1], route[0], conn.Stroke, conn.Opacity)
	}

	drawEdgeLabel(dc, conn, route)
}

// drawArrowhead draws a filled triangle at `tip`, pointing away from `from`.
// D2 has several arrowhead shapes; we render every non-"none" arrowhead as a
// filled triangle, which is D2's own default (d2target.DefaultArrowhead).
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

func drawEdgeLabel(dc *gg.Context, conn d2target.Connection, route []*geo.Point) {
	if conn.Label == "" {
		return
	}

	mx, my := midpoint(route)

	// D2's own SVG renderer masks the connection's stroke out behind the
	// label (d2svg's makeLabelMask) rather than drawing text directly over
	// the line, so the line visually gaps around the label instead of
	// running through the glyphs. We approximate that by painting an opaque
	// page-background rect first. This can look wrong if the label happens
	// to sit over another shape's differently-colored fill rather than bare
	// canvas, but that's an uncommon case; matching the common one (label on
	// top of just the line) is the point.
	labelW, labelH := float64(conn.LabelWidth), float64(conn.LabelHeight)
	dc.SetHexColor(theme.Neutrals.N7)
	dc.DrawRectangle(mx-labelW/2-2, my-labelH/2, labelW+4, labelH)
	dc.Fill()

	dc.SetFontFace(fontFace(conn.Bold, float64(conn.FontSize)))
	setColor(dc, conn.GetFontColor(), 1)
	drawMultilineAt(dc, conn.Label, mx, my)
}

// midpoint returns the point halfway along the route's total path length.
func midpoint(route []*geo.Point) (x, y float64) {
	total := 0.0
	for i := 1; i < len(route); i++ {
		total += geo.EuclideanDistance(route[i-1].X, route[i-1].Y, route[i].X, route[i].Y)
	}

	target := total / 2
	walked := 0.0
	for i := 1; i < len(route); i++ {
		seg := geo.EuclideanDistance(route[i-1].X, route[i-1].Y, route[i].X, route[i].Y)
		if walked+seg >= target {
			t := 0.0
			if seg > 0 {
				t = (target - walked) / seg
			}
			return route[i-1].X + t*(route[i].X-route[i-1].X),
				route[i-1].Y + t*(route[i].Y-route[i-1].Y)
		}
		walked += seg
	}
	last := route[len(route)-1]
	return last.X, last.Y
}
