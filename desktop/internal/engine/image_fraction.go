package engine

import (
	"math"
	"strings"
	"unicode"
)

// Fraction layout uses the same pen glyphs as ordinary text, with a shared
// baseline and explicit boxes above/below the bar. No raster tracing is used.
type imageMathBox struct {
	width, top, bottom float64
	strokes            []Stroke
}

type imageMathParser struct {
	font           *Font
	text           []rune
	pos, fractions int
}

func (p *imageMathParser) sequence(depth int, grouped bool) (imageMathBox, bool) {
	if depth > 4 {
		return imageMathBox{}, false
	}
	out := imageMathBox{top: math.Inf(1), bottom: math.Inf(-1)}
	gap := p.font.Units * .16
	for p.pos < len(p.text) && p.text[p.pos] != '}' {
		var item imageMathBox
		var ok bool
		if strings.HasPrefix(string(p.text[p.pos:]), `\frac{`) {
			p.fractions++
			if p.fractions > 4 {
				return imageMathBox{}, false
			}
			p.pos += 5
			numerator, valid := p.group(depth + 1)
			if !valid {
				return imageMathBox{}, false
			}
			denominator, valid := p.group(depth + 1)
			if !valid {
				return imageMathBox{}, false
			}
			unit := p.font.Units
			// A fraction bar sits on the equation's math axis. Leave room for
			// numerator descenders/subscripts and denominator ascenders.
			axis, space, padding := -unit*.3, unit*.18, unit*.16
			if equal, ok := p.glyph('='); ok {
				axis = (equal.top + equal.bottom) / 2
			}
			item.width = math.Max(numerator.width, denominator.width) + padding*2
			ny, dy := axis-space-numerator.bottom, axis+space-denominator.top
			item.top, item.bottom = numerator.top+ny, denominator.bottom+dy
			item.strokes = appendMathStrokes(item.strokes, numerator.strokes, (item.width-numerator.width)/2, ny, 1)
			item.strokes = appendMathStrokes(item.strokes, denominator.strokes, (item.width-denominator.width)/2, dy, 1)
			item.strokes = append(item.strokes, Stroke{{0, axis}, {item.width, axis}})
			ok = true
		} else {
			r := p.text[p.pos]
			if !imageMathLetter(r) {
				return imageMathBox{}, false
			}
			p.pos++
			item, ok = p.glyph(r)
			if ok && p.pos < len(p.text) && (p.text[p.pos] == '_' || p.text[p.pos] == '^') {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					return imageMathBox{}, false
				}
				kind := p.text[p.pos]
				p.pos++
				start := p.pos
				script, valid := p.group(depth + 1)
				if !valid {
					return imageMathBox{}, false
				}
				// Scripts are deliberately limited to the existing flat notation.
				for _, c := range p.text[start+1 : p.pos-1] {
					if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '+' && c != '-' {
						return imageMathBox{}, false
					}
				}
				scale := .65
				// Match ordinary image-math script offsets, translated from
				// font coordinates into this parser's baseline coordinates.
				shift := p.font.Units * (.4 - (1 - scale))
				if kind == '^' {
					shift = p.font.Units * (-.4 - (1 - scale))
				}
				item.strokes = appendMathStrokes(item.strokes, script.strokes, item.width+gap*.3, shift, scale)
				item.width += gap*.3 + script.width*scale
				item.top = math.Min(item.top, script.top*scale+shift)
				item.bottom = math.Max(item.bottom, script.bottom*scale+shift)
			}
		}
		if !ok {
			return imageMathBox{}, false
		}
		out.strokes = appendMathStrokes(out.strokes, item.strokes, out.width, 0, 1)
		out.width += item.width + gap
		out.top = math.Min(out.top, item.top)
		out.bottom = math.Max(out.bottom, item.bottom)
	}
	if !finite(out.top) {
		return imageMathBox{}, false
	}
	out.width -= gap
	if grouped {
		if p.pos >= len(p.text) || p.text[p.pos] != '}' {
			return imageMathBox{}, false
		}
		p.pos++
	}
	return out, true
}

func (p *imageMathParser) group(depth int) (imageMathBox, bool) {
	if p.pos >= len(p.text) || p.text[p.pos] != '{' {
		return imageMathBox{}, false
	}
	p.pos++
	return p.sequence(depth, true)
}

func imageMathLetter(r rune) bool {
	return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r >= 'α' && r <= 'ω' || r >= 'Α' && r <= 'Ω' || strings.ContainsRune("+-=().,/'·", r)
}

func (p *imageMathParser) glyph(r rune) (imageMathBox, bool) {
	g, ok := p.font.Glyphs[r]
	if !ok {
		return imageMathBox{}, false
	}
	left, top, right, bottom := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range g.Strokes {
		for _, v := range s {
			left = math.Min(left, v.X)
			right = math.Max(right, v.X)
			top = math.Min(top, v.Y)
			bottom = math.Max(bottom, v.Y)
		}
	}
	if !finite(left) {
		return imageMathBox{}, false
	}
	// The built-in font uses Units as its writing baseline. Preserve each
	// glyph's original dimensions, including narrow I, operators and descenders.
	baseline := p.font.Units
	box := imageMathBox{width: right - left, top: top - baseline, bottom: bottom - baseline}
	box.strokes = appendMathStrokes(nil, g.Strokes, -left, -baseline, 1)
	return box, true
}

func appendMathStrokes(dst []Stroke, src []Stroke, x, y, scale float64) []Stroke {
	for _, s := range src {
		stroke := make(Stroke, len(s))
		for i, v := range s {
			stroke[i] = Point{x + v.X*scale, y + v.Y*scale}
		}
		dst = append(dst, stroke)
	}
	return dst
}

func imageFractionStrokes(f *Font, box OCRBox, asset ImageAsset) []Stroke {
	if len([]rune(box.Text)) > 160 {
		return nil
	}
	p := imageMathParser{font: f, text: []rune(box.Text)}
	layout, ok := p.sequence(0, false)
	if !ok || p.pos != len(p.text) || p.fractions == 0 || layout.width <= 0 || layout.bottom <= layout.top {
		return nil
	}
	scale := math.Min(box.W*asset.Width/layout.width, box.H*asset.Height/(layout.bottom-layout.top))
	x := asset.X + box.X*asset.Width + (box.W*asset.Width-layout.width*scale)/2
	y := asset.Y + box.Y*asset.Height + (box.H*asset.Height-(layout.bottom-layout.top)*scale)/2 - layout.top*scale
	strokes := appendMathStrokes(nil, layout.strokes, x, y, scale)
	for _, s := range strokes {
		for i, v := range s {
			s[i] = imagePoint(asset, v.X, v.Y)
		}
	}
	return strokes
}
