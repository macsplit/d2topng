package render

import (
	"github.com/fogleman/gg"
	"oss.terrastruct.com/d2/d2target"
	"oss.terrastruct.com/d2/lib/geo"
	"oss.terrastruct.com/d2/lib/label"
	"oss.terrastruct.com/d2/lib/shape"
	"oss.terrastruct.com/d2/lib/svg"
)

// Shape types with their own bespoke SVG rendering in d2svg (composite
// multi-element drawings, not a single filled/stroked path) rather than a
// plain shape.NewShape(...).GetSVGPathData() outline. Rendered as a plain
// rounded rectangle for now.
var compositeShapeTypes = map[string]bool{
	d2target.ShapeClass:    true,
	d2target.ShapeSQLTable: true,
	d2target.ShapeImage:    true,
}

func isRectLike(shapeType string) bool {
	return shapeType == d2target.ShapeRectangle || shapeType == d2target.ShapeSequenceDiagram ||
		shapeType == d2target.ShapeHierarchy || shapeType == "" || compositeShapeTypes[shapeType]
}

func drawShape(dc *gg.Context, s d2target.Shape) {
	x, y := float64(s.Pos.X), float64(s.Pos.Y)
	w, h := float64(s.Width), float64(s.Height)

	if s.Shadow {
		drawShadow(dc, s, x, y, w, h)
	}

	if s.Multiple {
		mx, my := x+d2target.MULTIPLE_OFFSET, y-d2target.MULTIPLE_OFFSET
		drawOutline(dc, s, mx, my, w, h, s.Fill, s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)
	}

	if s.ThreeDee && isRectLike(s.Type) {
		draw3D(dc, s, x, y, w, h)
	} else {
		drawOutline(dc, s, x, y, w, h, s.Fill, s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)
	}

	if s.DoubleBorder {
		const inset = d2target.INNER_BORDER_OFFSET
		drawOutline(dc, s, x+inset, y+inset, w-2*inset, h-2*inset, "none", s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)
	}

	drawLabel(dc, s, x, y, w, h)
}

// drawOutline traces and fills/strokes shape s's outline at an explicit
// (possibly offset/inset) box, with an explicit fill/stroke. Reused for the
// shape's main body as well as its shadow, "multiple", and double-border
// decorations, so those work uniformly across every shape type.
func drawOutline(dc *gg.Context, s d2target.Shape, x, y, w, h float64, fill, stroke string, strokeWidth int, strokeDash, opacity float64) {
	switch {
	case isRectLike(s.Type):
		traceRect(dc, s.BorderRadius, x, y, w, h)
		fillAndStroke(dc, fill, stroke, strokeWidth, strokeDash, opacity)
	case s.Type == d2target.ShapeOval:
		// Oval (and circle, which the compiler normalizes to an oval with
		// width==height) has no path geometry in lib/shape — D2 itself draws
		// it as a native SVG <ellipse>, so we do the same rather than routing
		// through the generic path branch, which would silently draw nothing.
		dc.NewSubPath()
		dc.DrawEllipse(x+w/2, y+h/2, w/2, h/2)
		fillAndStroke(dc, fill, stroke, strokeWidth, strokeDash, opacity)
	default:
		shapeType, ok := d2target.DSL_SHAPE_TO_SHAPE_TYPE[s.Type]
		if !ok {
			traceRect(dc, s.BorderRadius, x, y, w, h)
			fillAndStroke(dc, fill, stroke, strokeWidth, strokeDash, opacity)
			return
		}
		box := geo.NewBox(geo.NewPoint(x, y), w, h)
		shp := shape.NewShape(shapeType, box)
		for _, pathData := range shp.GetSVGPathData() {
			drawSVGPath(dc, pathData)
			fillAndStroke(dc, fill, stroke, strokeWidth, strokeDash, opacity)
		}
	}
}

func traceRect(dc *gg.Context, borderRadius int, x, y, w, h float64) {
	if borderRadius > 0 {
		dc.DrawRoundedRectangle(x, y, w, h, float64(borderRadius))
	} else {
		dc.DrawRectangle(x, y, w, h)
	}
}

// drawShadow approximates D2's drop-shadow filter with an offset, solid,
// semi-transparent copy of the shape drawn before the real one.
func drawShadow(dc *gg.Context, s d2target.Shape, x, y, w, h float64) {
	sx, sy := x+d2target.SHADOW_SIZE_X, y+d2target.SHADOW_SIZE_Y
	drawOutline(dc, s, sx, sy, w, h, "#0A0F25", "none", 0, 0, 0.25)
}

// draw3D approximates D2's isometric-look 3D rectangle: a copy offset up and
// to the right, connected to the front face by two quadrilateral side/top
// panels, all in the shape's own fill/stroke.
func draw3D(dc *gg.Context, s d2target.Shape, x, y, w, h float64) {
	const off = d2target.THREE_DEE_OFFSET

	// Back copy.
	traceRect(dc, s.BorderRadius, x+off, y-off, w, h)
	fillAndStroke(dc, s.Fill, s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)

	// Top panel connecting front-top edge to back-top edge.
	dc.NewSubPath()
	dc.MoveTo(x, y)
	dc.LineTo(x+off, y-off)
	dc.LineTo(x+w+off, y-off)
	dc.LineTo(x+w, y)
	dc.ClosePath()
	fillAndStroke(dc, s.Fill, s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)

	// Right panel connecting front-right edge to back-right edge.
	dc.NewSubPath()
	dc.MoveTo(x+w, y)
	dc.LineTo(x+w+off, y-off)
	dc.LineTo(x+w+off, y-off+h)
	dc.LineTo(x+w, y+h)
	dc.ClosePath()
	fillAndStroke(dc, s.Fill, s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)

	// Front face on top.
	traceRect(dc, s.BorderRadius, x, y, w, h)
	fillAndStroke(dc, s.Fill, s.Stroke, s.StrokeWidth, s.StrokeDash, s.Opacity)
}

func fillAndStroke(dc *gg.Context, fill, stroke string, strokeWidth int, strokeDash, opacity float64) {
	hasFill := setColor(dc, fill, opacity)
	hasStroke := strokeWidth > 0 && stroke != "" && stroke != "none"

	if hasFill && !hasStroke {
		dc.Fill()
		return
	}
	if hasFill {
		dc.FillPreserve()
	}
	if hasStroke {
		// Stroke width and dash lengths are applied in device pixels by gg
		// (unlike shape geometry, which the transform matrix scales
		// automatically), so outputScale must be applied here explicitly.
		scaledWidth := float64(strokeWidth) * outputScale
		setColor(dc, stroke, opacity)
		dc.SetLineWidth(scaledWidth)
		if strokeDash > 0 {
			dashSize, gapSize := svg.GetStrokeDashAttributes(scaledWidth, strokeDash)
			dc.SetDash(dashSize, gapSize)
		} else {
			dc.SetDash()
		}
		dc.Stroke()
		return
	}
	if !hasFill {
		dc.ClearPath()
	}
}

func drawLabel(dc *gg.Context, s d2target.Shape, x, y, w, h float64) {
	if s.Label == "" {
		return
	}

	labelW, labelH := float64(s.LabelWidth), float64(s.LabelHeight)
	box := geo.NewBox(geo.NewPoint(x, y), w, h)
	pos := label.FromString(s.LabelPosition)
	tl := pos.GetPointOnBox(box, label.PADDING, labelW, labelH)

	dc.SetFontFace(fontFace(s.Bold, float64(s.FontSize)))
	setColor(dc, s.GetFontColor(), 1)
	drawWrappedAt(dc, s.Label, tl.X+labelW/2, tl.Y+labelH/2, labelW)
}

// drawWrappedAt draws word-wrapped text centered at a point given in the
// current (possibly scaled) coordinate space. gg measures word-wrap against
// the font face directly, ignoring the transform matrix, so — since
// fontFace() already bakes outputScale into the face's point size — position
// and wrap width must be resolved to the same device-pixel space before
// drawing, rather than left for the matrix to transform. We do that by
// transforming the point ourselves, then drawing under a temporarily
// identity matrix with the wrap width scaled to match.
func drawWrappedAt(dc *gg.Context, text string, x, y, width float64) {
	dx, dy := dc.TransformPoint(x, y)
	dc.Push()
	dc.Identity()
	dc.DrawStringWrapped(text, dx, dy, 0.5, 0.5, width*outputScale, 1.2, gg.AlignCenter)
	dc.Pop()
}
