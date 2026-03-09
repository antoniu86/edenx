package editor

import "github.com/gdamore/tcell/v2"

// Theme defines all color pairs used by the editor.
type Theme struct {
	Name        string
	EditorBg    tcell.Color
	EditorFg    tcell.Color
	LineNumBg   tcell.Color
	LineNumFg   tcell.Color
	StatusBg    tcell.Color
	StatusFg    tcell.Color
	TabBg       tcell.Color
	TabFg       tcell.Color
	ActiveTabBg tcell.Color
	ActiveTabFg tcell.Color
	SelectBg    tcell.Color
	SelectFg    tcell.Color
	SearchBg    tcell.Color
	SearchFg    tcell.Color
	CurMatchBg  tcell.Color
	CurMatchFg  tcell.Color
	NavBg       tcell.Color
	NavFg       tcell.Color
	NavSelBg    tcell.Color
	NavSelFg    tcell.Color
	NavBorderFg tcell.Color

	// Syntax highlight foreground colors.
	SynKeyword tcell.Color
	SynType    tcell.Color
	SynBuiltin tcell.Color
	SynString  tcell.Color
	SynComment tcell.Color
	SynNumber  tcell.Color

	// Bracket match highlight.
	BracketBg tcell.Color
	BracketFg tcell.Color
}

// synColor maps a SynToken to its foreground color for this theme.
func (t Theme) synColor(tok SynToken) tcell.Color {
	switch tok {
	case SynKeyword:
		return t.SynKeyword
	case SynType:
		return t.SynType
	case SynBuiltin:
		return t.SynBuiltin
	case SynString:
		return t.SynString
	case SynComment:
		return t.SynComment
	case SynNumber:
		return t.SynNumber
	default:
		return t.EditorFg
	}
}

// ThemeOrder defines the F2 cycle order.
var ThemeOrder = []string{"default", "green", "dark", "light", "monokai"}

var themes = map[string]Theme{
	"default": {
		Name: "default",
		EditorBg: tcell.ColorDefault, EditorFg: tcell.ColorDefault,
		LineNumBg: tcell.ColorDefault, LineNumFg: tcell.ColorDarkGray,
		StatusBg: tcell.ColorSilver, StatusFg: tcell.ColorBlack,
		TabBg: tcell.ColorDefault, TabFg: tcell.ColorDarkGray,
		ActiveTabBg: tcell.ColorSilver, ActiveTabFg: tcell.ColorBlack,
		SelectBg: tcell.ColorNavy, SelectFg: tcell.ColorWhite,
		SearchBg: tcell.ColorOlive, SearchFg: tcell.ColorBlack,
		CurMatchBg: tcell.ColorYellow, CurMatchFg: tcell.ColorBlack,
		NavBg: tcell.ColorDefault, NavFg: tcell.ColorDefault,
		NavSelBg: tcell.ColorNavy, NavSelFg: tcell.ColorWhite,
		NavBorderFg: tcell.ColorDarkGray,
		SynKeyword: tcell.ColorBlue, SynType: tcell.ColorTeal,
		SynBuiltin: tcell.ColorNavy, SynString: tcell.ColorGreen,
		SynComment: tcell.ColorDarkGray, SynNumber: tcell.ColorMaroon,
		BracketBg: tcell.ColorTeal, BracketFg: tcell.ColorWhite,
	},
	"green": {
		Name: "green",
		EditorBg: tcell.NewRGBColor(0, 40, 0), EditorFg: tcell.NewRGBColor(0, 220, 0),
		LineNumBg: tcell.NewRGBColor(0, 25, 0), LineNumFg: tcell.NewRGBColor(0, 140, 0),
		StatusBg: tcell.NewRGBColor(0, 80, 0), StatusFg: tcell.NewRGBColor(0, 255, 0),
		TabBg: tcell.NewRGBColor(0, 20, 0), TabFg: tcell.NewRGBColor(0, 120, 0),
		ActiveTabBg: tcell.NewRGBColor(0, 80, 0), ActiveTabFg: tcell.NewRGBColor(0, 255, 0),
		SelectBg: tcell.NewRGBColor(0, 110, 0), SelectFg: tcell.NewRGBColor(200, 255, 200),
		SearchBg: tcell.NewRGBColor(60, 90, 0), SearchFg: tcell.ColorBlack,
		CurMatchBg: tcell.NewRGBColor(100, 180, 0), CurMatchFg: tcell.ColorBlack,
		NavBg: tcell.NewRGBColor(0, 20, 0), NavFg: tcell.NewRGBColor(0, 200, 0),
		NavSelBg: tcell.NewRGBColor(0, 100, 0), NavSelFg: tcell.NewRGBColor(200, 255, 200),
		NavBorderFg: tcell.NewRGBColor(0, 140, 0),
		SynKeyword: tcell.NewRGBColor(0, 255, 0), SynType: tcell.NewRGBColor(100, 255, 150),
		SynBuiltin: tcell.NewRGBColor(0, 200, 100), SynString: tcell.NewRGBColor(150, 230, 100),
		SynComment: tcell.NewRGBColor(0, 100, 0), SynNumber: tcell.NewRGBColor(100, 220, 180),
		BracketBg: tcell.NewRGBColor(0, 150, 80), BracketFg: tcell.NewRGBColor(200, 255, 200),
	},
	"dark": {
		Name: "dark",
		EditorBg: tcell.NewRGBColor(30, 30, 30), EditorFg: tcell.NewRGBColor(220, 220, 220),
		LineNumBg: tcell.NewRGBColor(20, 20, 20), LineNumFg: tcell.NewRGBColor(100, 100, 100),
		StatusBg: tcell.NewRGBColor(55, 55, 55), StatusFg: tcell.NewRGBColor(220, 220, 220),
		TabBg: tcell.NewRGBColor(20, 20, 20), TabFg: tcell.NewRGBColor(140, 140, 140),
		ActiveTabBg: tcell.NewRGBColor(55, 55, 55), ActiveTabFg: tcell.NewRGBColor(255, 255, 255),
		SelectBg: tcell.NewRGBColor(65, 105, 150), SelectFg: tcell.NewRGBColor(255, 255, 255),
		SearchBg: tcell.NewRGBColor(100, 80, 0), SearchFg: tcell.ColorBlack,
		CurMatchBg: tcell.NewRGBColor(200, 160, 0), CurMatchFg: tcell.ColorBlack,
		NavBg: tcell.NewRGBColor(20, 20, 20), NavFg: tcell.NewRGBColor(200, 200, 200),
		NavSelBg: tcell.NewRGBColor(65, 105, 150), NavSelFg: tcell.NewRGBColor(255, 255, 255),
		NavBorderFg: tcell.NewRGBColor(80, 80, 80),
		SynKeyword: tcell.NewRGBColor(86, 156, 214), SynType: tcell.NewRGBColor(78, 201, 176),
		SynBuiltin: tcell.NewRGBColor(220, 220, 170), SynString: tcell.NewRGBColor(206, 145, 120),
		SynComment: tcell.NewRGBColor(106, 153, 85), SynNumber: tcell.NewRGBColor(181, 206, 168),
		BracketBg: tcell.NewRGBColor(60, 60, 20), BracketFg: tcell.NewRGBColor(255, 230, 100),
	},
	"light": {
		Name: "light",
		EditorBg: tcell.NewRGBColor(252, 252, 252), EditorFg: tcell.NewRGBColor(30, 30, 30),
		LineNumBg: tcell.NewRGBColor(235, 235, 235), LineNumFg: tcell.NewRGBColor(160, 160, 160),
		StatusBg: tcell.NewRGBColor(210, 210, 210), StatusFg: tcell.NewRGBColor(30, 30, 30),
		TabBg: tcell.NewRGBColor(220, 220, 220), TabFg: tcell.NewRGBColor(90, 90, 90),
		ActiveTabBg: tcell.NewRGBColor(252, 252, 252), ActiveTabFg: tcell.NewRGBColor(0, 0, 0),
		SelectBg: tcell.NewRGBColor(180, 210, 245), SelectFg: tcell.NewRGBColor(0, 0, 0),
		SearchBg: tcell.NewRGBColor(255, 225, 100), SearchFg: tcell.NewRGBColor(0, 0, 0),
		CurMatchBg: tcell.NewRGBColor(255, 140, 0), CurMatchFg: tcell.ColorBlack,
		NavBg: tcell.NewRGBColor(235, 235, 235), NavFg: tcell.NewRGBColor(30, 30, 30),
		NavSelBg: tcell.NewRGBColor(180, 210, 245), NavSelFg: tcell.NewRGBColor(0, 0, 0),
		NavBorderFg: tcell.NewRGBColor(160, 160, 160),
		SynKeyword: tcell.NewRGBColor(0, 0, 255), SynType: tcell.NewRGBColor(38, 127, 153),
		SynBuiltin: tcell.NewRGBColor(121, 94, 38), SynString: tcell.NewRGBColor(163, 21, 21),
		SynComment: tcell.NewRGBColor(0, 128, 0), SynNumber: tcell.NewRGBColor(9, 134, 88),
		BracketBg: tcell.NewRGBColor(200, 230, 180), BracketFg: tcell.NewRGBColor(0, 0, 0),
	},
	"monokai": {
		Name: "monokai",
		EditorBg: tcell.NewRGBColor(39, 40, 34), EditorFg: tcell.NewRGBColor(248, 248, 242),
		LineNumBg: tcell.NewRGBColor(30, 31, 26), LineNumFg: tcell.NewRGBColor(117, 113, 94),
		StatusBg: tcell.NewRGBColor(75, 70, 58), StatusFg: tcell.NewRGBColor(248, 248, 242),
		TabBg: tcell.NewRGBColor(30, 31, 26), TabFg: tcell.NewRGBColor(117, 113, 94),
		ActiveTabBg: tcell.NewRGBColor(75, 70, 58), ActiveTabFg: tcell.NewRGBColor(248, 248, 242),
		SelectBg: tcell.NewRGBColor(73, 72, 62), SelectFg: tcell.NewRGBColor(248, 248, 242),
		SearchBg: tcell.NewRGBColor(80, 70, 30), SearchFg: tcell.NewRGBColor(248, 248, 242),
		CurMatchBg: tcell.NewRGBColor(230, 219, 116), CurMatchFg: tcell.NewRGBColor(39, 40, 34),
		NavBg: tcell.NewRGBColor(30, 31, 26), NavFg: tcell.NewRGBColor(248, 248, 242),
		NavSelBg: tcell.NewRGBColor(73, 72, 62), NavSelFg: tcell.NewRGBColor(248, 248, 242),
		NavBorderFg: tcell.NewRGBColor(117, 113, 94),
		SynKeyword: tcell.NewRGBColor(249, 38, 114), SynType: tcell.NewRGBColor(102, 217, 232),
		SynBuiltin: tcell.NewRGBColor(166, 226, 46), SynString: tcell.NewRGBColor(230, 219, 116),
		SynComment: tcell.NewRGBColor(117, 113, 94), SynNumber: tcell.NewRGBColor(174, 129, 255),
		BracketBg: tcell.NewRGBColor(80, 90, 40), BracketFg: tcell.NewRGBColor(248, 248, 242),
	},
}

// GetTheme returns a theme by name, falling back to "default".
func GetTheme(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["default"]
}

// NextTheme returns the next theme name in the cycle.
func NextTheme(current string) string {
	for i, name := range ThemeOrder {
		if name == current {
			return ThemeOrder[(i+1)%len(ThemeOrder)]
		}
	}
	return ThemeOrder[0]
}
