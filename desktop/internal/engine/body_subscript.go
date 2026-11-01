package engine

import (
	"strings"
	"unicode"
)

func bodySubscriptEnd(runes []rune, start int) int {
	if start+4 >= len(runes) || unicode.IsSpace(runes[start]) || strings.ContainsRune("_{}\\^", runes[start]) || runes[start+1] != '_' || runes[start+2] != '{' {
		return 0
	}
	nonspace := false
	for i := start + 3; i < len(runes); i++ {
		switch runes[i] {
		case '{', '\\', '_', '^':
			return 0
		case '}':
			if nonspace {
				return i
			}
			return 0
		}
		nonspace = nonspace || !unicode.IsSpace(runes[i])
	}
	return 0
}

func (f *Font) bodyScriptSupported(base rune, sub []rune) bool {
	if _, ok := f.Glyphs[base]; !ok {
		return false
	}
	for _, r := range sub {
		if _, ok := f.Glyphs[r]; !ok && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func (f *Font) bodyScriptGlyph(base rune, sub []rune, s LayoutSettings) (Glyph, float64) {
	glyph := Glyph{}
	x := 0.0
	appendGlyph := func(r rune, scale, shift float64) {
		style := s
		style.FontSize *= scale
		advance := f.advance(r, style)
		if g, ok := f.Glyphs[r]; ok && !unicode.IsSpace(r) {
			bearing := textGlyphBearing(g, f.Units, advance, style)
			glyph.Strokes = appendMathStrokes(glyph.Strokes, g.Strokes, (x+bearing)/s.FontSize*f.Units, shift*f.Units, scale)
		}
		x += advance
	}
	appendGlyph(base, 1, 0)
	for _, r := range sub {
		appendGlyph(r, .65, .4)
	}
	glyph.Width = (x - s.LetterGap) / s.FontSize * f.Units
	return glyph, x
}
