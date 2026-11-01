package engine

import "unicode"

// A font's em includes whitespace above and below its ink. Check the actual
// strokes so 110% type can fit a row without shrinking or stretching glyphs.
func (f *Font) lineInkBounds(runes []rune, size float64) (top, bottom float64, hasInk bool) {
	for _, r := range runes {
		if unicode.IsSpace(r) {
			continue
		}
		g, ok := f.Glyphs[r]
		if !ok {
			// Missing-glyph markers still reserve their full em square.
			top, bottom, hasInk = 0, max(bottom, size), true
			continue
		}
		for _, stroke := range g.Strokes {
			for _, p := range stroke {
				y := p.Y / f.Units * size
				if !hasInk {
					top, bottom, hasInk = y, y, true
				} else {
					top, bottom = min(top, y), max(bottom, y)
				}
			}
		}
	}
	return
}
