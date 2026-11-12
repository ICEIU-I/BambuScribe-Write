package engine

import "strings"

// Unicode Roman numerals remain one source character. Reuse the selected
// font's Latin strokes so rendering never rewrites formula IDs or edit ranges.
func addRomanNumerals(f *Font) {
	letters := []string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII", "L", "C", "D", "M"}
	for i, text := range letters {
		for _, entry := range []struct {
			char rune
			text string
		}{{rune(0x2160 + i), text}, {rune(0x2170 + i), strings.ToLower(text)}} {
			if _, exists := f.Glyphs[entry.char]; exists {
				continue
			}
			if len(entry.text) == 1 {
				if g, ok := f.Glyphs[rune(entry.text[0])]; ok {
					f.Glyphs[entry.char] = g
				}
				continue
			}
			if g, ok := composeRomanNumeral(f, entry.text); ok {
				f.Glyphs[entry.char] = g
			}
		}
	}
}

func composeRomanNumeral(f *Font, text string) (Glyph, bool) {
	var strokes []Stroke
	width := 0.0
	for i, r := range text {
		g, ok := f.Glyphs[r]
		if !ok {
			return Glyph{}, false
		}
		left, _, right, _, ok := inkBounds(g.Strokes)
		if !ok {
			return Glyph{}, false
		}
		if i > 0 {
			width += f.Units * .1
		}
		strokes = appendMathStrokes(strokes, g.Strokes, width-left, 0, 1)
		width += right - left
	}
	// Shrink only when the full group exceeds the cell. Both axes use the
	// same factor, preserving the letter shapes and shared baseline.
	scale := min(1, f.Units/width)
	strokes = appendMathStrokes(nil, strokes, 0, f.Units*(1-scale), scale)
	return Glyph{Width: width * scale, Strokes: strokes}, true
}
