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

	mx, my, nx, ny := midpointAndNormal(route)

	// Offset the label off to the side of the line rather than centering it
	// directly on top of the stroke. D2's own SVG renderer instead leaves the
	// label on the line and masks the stroke out from behind it, which only
	// looks right when there's bare canvas behind the label — if the label
	// happens to land on another shape's differently-colored fill, a mask
	// (or an opaque rect standing in for one, without real SVG masking)
	// paints over that fill incorrectly. Spacing the label away from the
	// line entirely sidesteps that, at the cost of no longer matching D2's
	// exact on-line placement.
	const gap = 4.0
	labelW, labelH := float64(conn.LabelWidth), float64(conn.LabelHeight)
	offset := float64(conn.StrokeWidth)/2 + gap
	lx := mx + nx*(offset+labelW/2)
	ly := my + ny*(offset+labelH/2)

	dc.SetFontFace(fontFace(conn.Bold, float64(conn.FontSize)))
	setColor(dc, conn.GetFontColor(), 1)
	drawMultilineAt(dc, conn.Label, lx, ly)
}

// midpointAndNormal returns the point halfway along the route's total path
// length, plus the unit vector normal to the segment it falls on. The normal
// always points "up" (toward decreasing Y, since D2's Y axis increases
// downward) rather than alternating with the segment's direction of travel,
// so labels land on a consistent side of the line instead of flipping
// between above and below depending on which way a given edge happens to be
// routed.
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

// midpoint returns just the point component of midpointAndNormal.
func midpoint(route []*geo.Point) (x, y float64) {
	x, y, _, _ = midpointAndNormal(route)
	return x, y
}

// segmentNormal returns the unit vector perpendicular to a->b, oriented
// toward decreasing Y ("up"). Degenerate (zero-length) segments fall back to
// straight up rather than dividing by zero.
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
