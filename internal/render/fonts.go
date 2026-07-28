package render

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"oss.terrastruct.com/d2/d2renderers/d2fonts"
)

// We reuse D2's own embedded SourceSansPro font bytes rather than shipping a
// separate font file, so visual output stays consistent with D2 and no extra
// font licensing decision is needed.
var regularTTF, boldTTF *truetype.Font

func init() {
	regularBytes := d2fonts.FontFaces.Get(d2fonts.Font{Family: d2fonts.SourceSansPro, Style: d2fonts.FONT_STYLE_REGULAR})
	boldBytes := d2fonts.FontFaces.Get(d2fonts.Font{Family: d2fonts.SourceSansPro, Style: d2fonts.FONT_STYLE_BOLD})

	var err error
	regularTTF, err = truetype.Parse(regularBytes)
	if err != nil {
		panic(err)
	}
	boldTTF, err = truetype.Parse(boldBytes)
	if err != nil {
		panic(err)
	}
}

// fontFace loads a face at `points`, scaled by outputScale — gg's transform
// matrix scales path geometry automatically but not font rendering, so this
// is the one place that needs to account for it explicitly (see outputScale
// in render.go).
func fontFace(bold bool, points float64) font.Face {
	f := regularTTF
	if bold {
		f = boldTTF
	}
	return truetype.NewFace(f, &truetype.Options{
		Size: points * outputScale,
		DPI:  72,
	})
}
