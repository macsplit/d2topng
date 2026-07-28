package render

import "oss.terrastruct.com/d2/d2themes/d2themescatalog"

// theme is the single hardcoded palette this renderer supports (D2's default,
// "Neutral Default", theme ID 0). There is no theme selection.
var theme = d2themescatalog.NeutralDefault.Colors

// resolveColor turns a d2target color field into a hex string ready for gg.
// Shape/connection color fields come back from d2lib.Compile as theme-slot
// tokens ("B6", "N1", "AA4", ...) rather than literal colors, resolved
// upstream inside d2svg's theme CSS which we bypass entirely. If a .d2 file
// sets an explicit color (e.g. "red" or "#ff0000"), that string won't match
// any token below and is returned unchanged.
func resolveColor(token string) string {
	switch token {
	case "N1":
		return theme.Neutrals.N1
	case "N2":
		return theme.Neutrals.N2
	case "N3":
		return theme.Neutrals.N3
	case "N4":
		return theme.Neutrals.N4
	case "N5":
		return theme.Neutrals.N5
	case "N6":
		return theme.Neutrals.N6
	case "N7":
		return theme.Neutrals.N7
	case "B1":
		return theme.B1
	case "B2":
		return theme.B2
	case "B3":
		return theme.B3
	case "B4":
		return theme.B4
	case "B5":
		return theme.B5
	case "B6":
		return theme.B6
	case "AA2":
		return theme.AA2
	case "AA4":
		return theme.AA4
	case "AA5":
		return theme.AA5
	case "AB4":
		return theme.AB4
	case "AB5":
		return theme.AB5
	default:
		return token
	}
}
