package render

import (
	"strconv"
	"strings"

	"github.com/fogleman/gg"
)

// drawSVGPath replays a D2 lib/shape path string (the subset of the SVG path
// mini-language D2 itself emits: M, L, H, V, C, Z, always with absolute,
// space-separated numeric coordinates already resolved in diagram space) as
// gg drawing calls. This lets us reuse D2's own shape geometry
// (oss.terrastruct.com/d2/lib/shape) for every non-rectangular shape instead
// of hand-coding path math per shape type.
func drawSVGPath(dc *gg.Context, pathData string) {
	tokens := strings.Fields(pathData)
	var cx, cy float64
	i := 0

	next := func() float64 {
		v, _ := strconv.ParseFloat(tokens[i], 64)
		i++
		return v
	}

	dc.NewSubPath()
	for i < len(tokens) {
		cmd := tokens[i]
		i++
		switch cmd {
		case "M":
			cx, cy = next(), next()
			dc.MoveTo(cx, cy)
		case "L":
			cx, cy = next(), next()
			dc.LineTo(cx, cy)
		case "H":
			cx = next()
			dc.LineTo(cx, cy)
		case "V":
			cy = next()
			dc.LineTo(cx, cy)
		case "C":
			x1, y1, x2, y2, x3, y3 := next(), next(), next(), next(), next(), next()
			dc.CubicTo(x1, y1, x2, y2, x3, y3)
			cx, cy = x3, y3
		case "Z":
			dc.ClosePath()
		}
	}
}
