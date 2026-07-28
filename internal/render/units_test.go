package render

import (
	"testing"

	"github.com/fogleman/gg"
	"oss.terrastruct.com/d2/lib/geo"
)

func TestResolveColor(t *testing.T) {
	cases := map[string]string{
		"N1":      theme.Neutrals.N1,
		"B6":      theme.B6,
		"AA4":     theme.AA4,
		"#ff0000": "#ff0000", // explicit literal color passes through untouched
		"red":     "red",
		"none":    "none",
		"":        "",
	}
	for token, want := range cases {
		if got := resolveColor(token); got != want {
			t.Errorf("resolveColor(%q) = %q, want %q", token, got, want)
		}
	}
}

func TestSetColorNoneIsSkipped(t *testing.T) {
	dc := gg.NewContext(1, 1)
	if setColor(dc, "none", 1) {
		t.Error("setColor(\"none\") should return false")
	}
	if setColor(dc, "", 1) {
		t.Error("setColor(\"\") should return false")
	}
	if !setColor(dc, "N1", 1) {
		t.Error("setColor(\"N1\") should return true")
	}
}

func TestDrawSVGPathRoundTrip(t *testing.T) {
	// A closed unit square, expressed in the same M/L/H/V/C/Z subset D2's
	// lib/shape emits.
	dc := gg.NewContext(10, 10)
	drawSVGPath(dc, "M 0 0 L 10 0 V 10 H 0 Z")
	// Filling should not panic and should leave a non-empty path behind
	// (Fill clears it), i.e. this just exercises the parser end to end.
	dc.SetRGB(0, 0, 0)
	dc.Fill()
}

func TestMidpointHalfwayAlongPolyline(t *testing.T) {
	route := []*geo.Point{
		geo.NewPoint(0, 0),
		geo.NewPoint(10, 0),
		geo.NewPoint(10, 10),
	}
	// total length 20, midpoint at length 10 == the corner point (10, 0)
	x, y := midpoint(route)
	if x != 10 || y != 0 {
		t.Errorf("midpoint = (%v, %v), want (10, 0)", x, y)
	}
}

func TestMidpointSinglePoint(t *testing.T) {
	route := []*geo.Point{geo.NewPoint(5, 5)}
	x, y := midpoint(route)
	if x != 5 || y != 5 {
		t.Errorf("midpoint = (%v, %v), want (5, 5)", x, y)
	}
}
