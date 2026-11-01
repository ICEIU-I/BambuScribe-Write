package engine

// UTF-16 offsets match the native textarea's selectionStart/selectionEnd,
// including supplementary characters such as emoji and rare CJK characters.
type CaretStop struct {
	Index int     `json:"index"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

type GlyphCell struct {
	Start  int     `json:"start"`
	End    int     `json:"end"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func utf16Width(r rune) int {
	if r > 0xffff {
		return 2
	}
	return 1
}

func lineOrigin(s LayoutSettings, indent, available, used float64) float64 {
	x := s.MarginLeft + indent
	if s.Align == "center" {
		x += (available - used) / 2
	} else if s.Align == "right" {
		x += available - used
	}
	return x
}
