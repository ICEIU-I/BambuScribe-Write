package engine

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Formula text is stored in the body as a standalone \( ... \) paragraph.
// Geometry is derived from glyphs, never from a table cell's available area.
type formulaParser struct {
	variation handwritingVariation
	font      *Font
	input     []rune
	pos       int
	ratio     float64
	nodes     int
}

func (p *formulaParser) group(depth int) (imageMathBox, error) {
	if p.pos >= len(p.input) || p.input[p.pos] != '{' {
		return imageMathBox{}, fmt.Errorf("公式缺少括号")
	}
	p.pos++
	b, err := p.sequence(depth+1, true)
	return b, err
}
func (p *formulaParser) sequence(depth int, grouped bool) (imageMathBox, error) {
	if depth > 8 {
		return imageMathBox{}, fmt.Errorf("公式嵌套过多，请拆分")
	}
	out := imageMathBox{top: math.Inf(1), bottom: math.Inf(-1)}
	unit := p.font.Units
	gap := .10 * unit
	lastGap := gap
	for p.pos < len(p.input) && p.input[p.pos] != '}' {
		atomIndex := p.pos
		p.nodes++
		if p.nodes > 2000 {
			return out, fmt.Errorf("公式过长，请拆分")
		}
		if unicode.IsSpace(p.input[p.pos]) {
			p.pos++
			continue
		}
		var b imageMathBox
		var err error
		tail := string(p.input[p.pos:])
		switch {
		case strings.HasPrefix(tail, `\frac`):
			p.pos += 5
			n, e := p.group(depth)
			if e != nil {
				return out, e
			}
			d, e := p.group(depth)
			if e != nil {
				return out, e
			}
			scale := p.ratio
			n = scaleFormulaBox(n, scale)
			d = scaleFormulaBox(d, scale)
			axis := p.axis()
			pad, space := unit*.08, unit*.10
			b.width = math.Max(n.width, d.width) + 2*pad
			ny, dy := axis-space-n.bottom, axis+space-d.top
			b.top, b.bottom = n.top+ny, d.bottom+dy
			b.strokes = appendMathStrokes(b.strokes, n.strokes, (b.width-n.width)/2, ny, 1)
			b.strokes = appendMathStrokes(b.strokes, d.strokes, (b.width-d.width)/2, dy, 1)
			b.strokes = append(b.strokes, Stroke{{0, axis}, {b.width, axis}})
		case strings.HasPrefix(tail, `\sqrt`):
			p.pos += 5
			b, err = p.group(depth)
			if err == nil {
				h := b.bottom - b.top
				w := unit * .45
				b.strokes = appendMathStrokes(nil, b.strokes, w, 0, 1)
				b.strokes = append(b.strokes, Stroke{{0, b.top + h*.65}, {w * .25, b.top + h*.6}, {w * .45, b.bottom}, {w * .8, b.top - unit*.1}, {w + b.width + unit*.08, b.top - unit*.1}})
				b.width += w + unit*.08
				b.top -= unit * .1
			}
		case strings.HasPrefix(tail, `\bar`):
			p.pos += 4
			b, err = p.group(depth)
			if err == nil {
				b.top -= unit * .10
				b.strokes = append(b.strokes, Stroke{{0, b.top}, {b.width, b.top}})
			}
		default:
			r := p.input[p.pos]
			p.pos++
			if r == '\\' || r == '{' || r == '_' || r == '^' || r == '\u0305' {
				return out, fmt.Errorf("公式符号格式无效，请检查分式或上下标")
			}
			var ok bool
			b, ok = (&imageMathParser{font: p.font}).glyph(r)
			b.strokes = p.variation.strokes(b.strokes, p.pos-1)
			if !ok {
				return out, fmt.Errorf("公式缺字：%c，请替换该符号", r)
			}
		}
		if err != nil {
			return out, err
		}
		// Scripts belong to their base glyph/group and use its nominal size.
		for p.pos < len(p.input) {
			r := p.input[p.pos]
			if r == '\u0305' {
				p.pos++
				b.top -= unit * .10
				b.strokes = append(b.strokes, Stroke{{0, b.top}, {b.width, b.top}})
				continue
			}
			if r != '_' && r != '^' {
				break
			}
			p.pos++
			script, e := p.group(depth)
			if e != nil {
				return out, e
			}
			script = scaleFormulaBox(script, .65)
			shift := unit * .12
			if r == '^' {
				shift = b.top - script.bottom + unit*.15
			}
			b.strokes = appendMathStrokes(b.strokes, script.strokes, b.width+gap*.25, shift, 1)
			b.width += gap*.25 + script.width
			b.top = math.Min(b.top, script.top+shift)
			b.bottom = math.Max(b.bottom, script.bottom+shift)
		}
		out.strokes = appendMathStrokes(out.strokes, b.strokes, out.width, 0, 1)
		lastGap = p.variation.gap(atomIndex, gap)
		out.width += b.width + lastGap
		out.top = math.Min(out.top, b.top)
		out.bottom = math.Max(out.bottom, b.bottom)
	}
	if !finite(out.top) {
		return out, fmt.Errorf("公式内容不能为空")
	}
	out.width -= lastGap
	if grouped {
		if p.pos >= len(p.input) || p.input[p.pos] != '}' {
			return out, fmt.Errorf("公式括号未闭合")
		}
		p.pos++
	}
	return out, nil
}
func (p *formulaParser) axis() float64 {
	b, ok := (&imageMathParser{font: p.font}).glyph('=')
	if ok {
		return (b.top + b.bottom) / 2
	}
	return -.3 * p.font.Units
}
func scaleFormulaBox(b imageMathBox, scale float64) imageMathBox {
	return imageMathBox{width: b.width * scale, top: b.top * scale, bottom: b.bottom * scale, strokes: appendMathStrokes(nil, b.strokes, 0, 0, scale)}
}
func (f *Font) formulaBox(text string, s LayoutSettings, variation ...handwritingVariation) (imageMathBox, error) {
	ratio := s.FractionScale
	if ratio == 0 {
		ratio = 85
	}
	p := formulaParser{font: f, input: []rune(text), ratio: ratio / 100}
	if len(variation) > 0 {
		p.variation = variation[0]
	}
	b, err := p.sequence(0, false)
	if err == nil && p.pos != len(p.input) {
		err = fmt.Errorf("公式括号不匹配")
	}
	return scaleFormulaBox(b, s.FontSize/f.Units), err
}
func formulaSource(paragraph string) (string, bool) {
	if strings.HasPrefix(paragraph, `\(`) && strings.HasSuffix(paragraph, `\)`) {
		return paragraph[2 : len(paragraph)-2], true
	}
	return "", false
}

// Split only between complete equality terms. Fractions never straddle lines.
func (f *Font) formulaLines(source string, s LayoutSettings, width float64, variation handwritingVariation) ([]formulaLine, error) {
	runes := []rune(source)
	starts := []int{0}
	depth := 0
	for i, r := range runes {
		if r == '{' {
			depth++
		}
		if r == '}' {
			depth--
		}
		if r == '=' && depth == 0 && i > 0 {
			starts = append(starts, i)
		}
	}
	starts = append(starts, len(runes))
	out := []formulaLine{}
	start, end := 0, 0
	var box imageMathBox
	for i := 0; i < len(starts)-1; i++ {
		next := starts[i+1]
		b, err := f.formulaBox(string(runes[start:next]), s, variation.at(start))
		if err != nil {
			return nil, err
		}
		if b.width > width+1e-8 && end > start {
			out = append(out, formulaLine{source: string(runes[start:end]), box: box})
			start = end
			b, err = f.formulaBox(string(runes[start:next]), s, variation.at(start))
			if err != nil {
				return nil, err
			}
		}
		if b.width > width+1e-8 {
			return nil, fmt.Errorf("单个分式超过正文宽度，请减小字号或拆分计算式")
		}
		box = b
		end = next
	}
	out = append(out, formulaLine{source: string(runes[start:end]), box: box})
	return out, nil
}

type formulaLine struct {
	source string
	box    imageMathBox
}
