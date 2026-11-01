package engine

import (
	"maps"
	"strings"
	"unicode"
)

// A base and its subscript share one wrapping/selection cell. Temporary glyphs
// exist only in this paragraph's font; saved text and editor indices retain the
// original source, including _{...}. Ordinary body layout owns the line grid.
type bodyScript struct {
	source  string
	base    rune
	advance float64
	length  int
}

func (f *Font) bodySource(r rune) string {
	if script, ok := f.bodyScripts[r]; ok {
		return script.source
	}
	return string(r)
}

func (f *Font) bodyRuneWidth(r rune) int {
	if script, ok := f.bodyScripts[r]; ok {
		return script.length
	}
	return utf16Width(r)
}

func (f *Font) bodyBase(r rune) rune {
	if script, ok := f.bodyScripts[r]; ok {
		return script.base
	}
	return r
}

func (f *Font) bodyRunes(text string, settings LayoutSettings) (*Font, []rune) {
	runes := []rune(text)
	if !strings.Contains(text, "_{") {
		return f, runes
	}
	font := f
	out := make([]rune, 0, len(runes))
	token := rune(0xf0000)
	for i := 0; i < len(runes); {
		end := bodySubscriptEnd(runes, i)
		if end == 0 || !f.bodyScriptSupported(runes[i], runes[i+3:end]) {
			// Preserve a malformed/unsupported group as one literal span. Do not
			// accidentally interpret a nested marker inside an invalid group.
			if i+2 < len(runes) && runes[i+1] == '_' && runes[i+2] == '{' {
				depth, next := 1, i+3
				for next < len(runes) && depth > 0 {
					if runes[next] == '{' {
						depth++
					}
					if runes[next] == '}' {
						depth--
					}
					next++
				}
				out = append(out, runes[i:next]...)
				i = next
				continue
			}
			out = append(out, runes[i])
			i++
			continue
		}
		if font == f {
			copy := *f
			copy.Glyphs = maps.Clone(f.Glyphs)
			copy.bodyScripts = make(map[rune]bodyScript)
			font = &copy
		}
		for {
			_, exists := font.Glyphs[token]
			if !exists && !strings.ContainsRune(text, token) {
				break
			}
			token++
		}
		source := string(runes[i : end+1])
		length := 0
		for _, r := range runes[i : end+1] {
			length += utf16Width(r)
		}
		glyph, advance := f.bodyScriptGlyph(runes[i], runes[i+3:end], settings)
		font.Glyphs[token] = glyph
		font.bodyScripts[token] = bodyScript{source, runes[i], advance, length}
		out = append(out, token)
		token++
		i = end + 1
	}
	return font, out
}

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
