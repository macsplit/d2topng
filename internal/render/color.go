package render

import (
	"github.com/fogleman/gg"
	"github.com/mazznoer/csscolorparser"
)

// setColor resolves a D2 color field (theme token, hex, or CSS color name)
// and applies it as dc's current color with opacity factored into alpha.
// Returns false for "" / "none" (D2's sentinel for "don't draw this"), in
// which case dc's color is left untouched and the caller should skip the
// fill/stroke.
func setColor(dc *gg.Context, token string, opacity float64) bool {
	if token == "" || token == "none" {
		return false
	}

	resolved := resolveColor(token)
	c, err := csscolorparser.Parse(resolved)
	if err != nil {
		dc.SetHexColor(resolved)
		return true
	}

	c.A *= opacity
	dc.SetColor(c)
	return true
}
