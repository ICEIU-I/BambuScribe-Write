package engine

import (
	"math"
	"unicode"
)

// Combining bars decorate the preceding character instead of occupying a cell.
// An unattached mark remains literal so missing content is still reported.
func appendTableCharacter(out []tableCharacter, r rune, scale, shift float64) []tableCharacter {
	if (r == '\u0305' || r == '\u0304') && len(out) > 0 {
		previous := &out[len(out)-1]
		if previous.scale == scale && previous.shift == shift && (unicode.IsLetter(previous.char) || unicode.IsDigit(previous.char)) {
			previous.overline = true
			return out
		}
	}
	return append(out, tableCharacter{char: r, scale: scale, shift: shift})
}

func (f *Font) tableOverline(g Glyph) Glyph {
	left, right, top, _ := tableGlyphBounds(g)
	if !finite(left) || right <= left {
		return g
	}
	// Copy the slice: embedded glyphs are shared by every renderer and goroutine.
	g.Strokes = append(append([]Stroke(nil), g.Strokes...), Stroke{{left, top - .12*f.Units}, {right, top - .12*f.Units}})
	return g
}

// Consecutive underscores denote a writable blank, drawn as one continuous
// stroke. Keep their original advance width, font size and subscript position.
func (f *Font) tableFillLine(line []tableLetter, x, y float64) (Stroke, int, float64) {
	first := line[0]
	if first.char != '_' || first.missing || len(first.glyph.Strokes) != 1 {
		return nil, 0, 0
	}
	count, advance := 0, 0.0
	for _, l := range line {
		if l.char != '_' || l.missing || l.size != first.size || l.shift != first.shift || len(l.glyph.Strokes) != 1 {
			break
		}
		count++
		advance += l.width
	}
	if count < 2 {
		return nil, 0, 0
	}
	left, right, top, bottom := tableGlyphBounds(first.glyph)
	if !finite(left) {
		return nil, 0, 0
	}
	last := line[count-1]
	x1 := x + first.bearing + left/f.Units*first.size
	x2 := x + advance - last.width + last.bearing + right/f.Units*last.size
	y += first.shift + (top+bottom)/2/f.Units*first.size
	return Stroke{{x1, y}, {x2, y}}, count, advance
}

func tableGlyphBounds(g Glyph) (left, right, top, bottom float64) {
	left, right, top, bottom = math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, stroke := range g.Strokes {
		for _, p := range stroke {
			left, right = min(left, p.X), max(right, p.X)
			top, bottom = min(top, p.Y), max(bottom, p.Y)
		}
	}
	return
}
