package engine

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// Body math and display formulas share grammar and geometric structures.
// Inline glyphs retain body advances and baseline; displays use compact ink.
type formulaParser struct {
	variation   handwritingVariation
	font        *Font
	input       []rune
	pos, nodes  int
	ratio       float64
	inline      bool
	hasFraction bool
	settings    LayoutSettings
}

func (p *formulaParser) group(depth int) (imageMathBox, error) {
	if p.pos >= len(p.input) || p.input[p.pos] != '{' {
		return imageMathBox{}, fmt.Errorf("公式缺少左花括号，请使用{内容}")
	}
	p.pos++
	return p.sequence(depth+1, '}')
}

func (p *formulaParser) sequence(depth int, closing rune) (imageMathBox, error) {
	if depth > 8 {
		return imageMathBox{}, fmt.Errorf("公式嵌套超过8层，请拆分")
	}
	out := imageMathBox{top: math.Inf(1), bottom: math.Inf(-1)}
	lastGap := 0.0
	for p.pos < len(p.input) && p.input[p.pos] != closing {
		if p.input[p.pos] == '}' {
			return out, fmt.Errorf("公式括号不匹配")
		}
		index := p.pos
		if unicode.IsSpace(p.input[p.pos]) {
			p.pos++
			if p.inline {
				out.width += p.font.advance(p.input[index], p.settings) / p.settings.FontSize * p.font.Units
			}
			continue
		}
		b, err := p.atom(depth)
		if err != nil {
			return out, err
		}
		out.strokes = appendMathStrokes(out.strokes, b.strokes, out.width, 0, 1)
		lastGap = p.gap(index)
		out.width += b.width + lastGap
		out.top, out.bottom = min(out.top, b.top), max(out.bottom, b.bottom)
	}
	if !finite(out.top) {
		return out, fmt.Errorf("公式内容不能为空")
	}
	out.width -= lastGap
	if closing != 0 {
		if p.pos >= len(p.input) || p.input[p.pos] != closing {
			return out, fmt.Errorf("公式括号未闭合，请补齐右括号")
		}
		p.pos++
	}
	return out, nil
}

func (p *formulaParser) gap(index int) float64 {
	if p.inline {
		return p.settings.LetterGap / p.settings.FontSize * p.font.Units
	}
	return p.variation.gap(index, .10*p.font.Units)
}

func (p *formulaParser) glyph(r rune, index int) (imageMathBox, error) {
	var b imageMathBox
	if p.inline {
		g, ok := p.font.Glyphs[r]
		if !ok {
			return b, fmt.Errorf("公式缺字：%c，请替换该符号", r)
		}
		advance := p.font.advance(r, p.settings)
		bearing := textGlyphBearing(g, p.font.Units, advance, p.settings) / p.settings.FontSize * p.font.Units
		b.width = (advance - p.settings.LetterGap) / p.settings.FontSize * p.font.Units
		b.strokes = appendMathStrokes(nil, g.Strokes, bearing, -p.font.Units, 1)
		_, _, b.top, b.bottom = tableGlyphBounds(Glyph{Strokes: b.strokes})
	} else {
		var ok bool
		b, ok = (&imageMathParser{font: p.font}).glyph(r)
		if !ok {
			return b, fmt.Errorf("公式缺字：%c，请替换该符号", r)
		}
	}
	b.strokes = p.variation.strokes(b.strokes, index)
	return b, nil
}

func (p *formulaParser) atom(depth int) (imageMathBox, error) {
	var b imageMathBox
	if depth > 8 {
		return b, fmt.Errorf("公式嵌套超过8层，请拆分")
	}
	p.nodes++
	if p.nodes > 2000 {
		return b, fmt.Errorf("公式过长，请拆分")
	}
	if p.pos >= len(p.input) {
		return b, fmt.Errorf("公式内容不能为空")
	}
	r, index := p.input[p.pos], p.pos
	var err error
	switch {
	case r == '\\':
		p.pos++
		if p.pos < len(p.input) && strings.ContainsRune(`_^{}\`, p.input[p.pos]) {
			b, err = p.glyph(p.input[p.pos], index)
			p.pos++
			break
		}
		start := p.pos
		for p.pos < len(p.input) && p.input[p.pos] >= 'a' && p.input[p.pos] <= 'z' {
			p.pos++
		}
		command := string(p.input[start:p.pos])
		switch command {
		case "frac":
			var n, d imageMathBox
			n, err = p.group(depth)
			if err == nil {
				d, err = p.group(depth)
			}
			if err == nil {
				b = p.fraction(n, d)
				p.hasFraction = true
			}
		case "sqrt":
			var degree *imageMathBox
			if p.pos < len(p.input) && p.input[p.pos] == '[' {
				p.pos++
				var d imageMathBox
				d, err = p.sequence(depth+1, ']')
				if err == nil {
					degree = &d
				}
			}
			if err == nil {
				b, err = p.group(depth)
			}
			if err == nil {
				b = p.radical(b, degree)
			}
		case "bar":
			b, err = p.group(depth)
			if err == nil {
				b = mathOverbar(b, p.font.Units)
			}
		default:
			return b, fmt.Errorf("不支持命令\\%s，请使用实际符号或查看输入规则", command)
		}
	case r == '√' && p.pos+1 < len(p.input) && strings.ContainsRune("(（{", p.input[p.pos+1]):
		open := p.input[p.pos+1]
		close := map[rune]rune{'(': ')', '（': '）', '{': '}'}[open]
		p.pos += 2
		b, err = p.sequence(depth+1, close)
		if err == nil {
			b = p.radical(b, nil)
		}
	case r == '{':
		b, err = p.group(depth)
	case r == '(' || r == '（':
		// A complete mathematical parenthesis group remains visible and can own scripts.
		closing := ')'
		if r == '（' {
			closing = '）'
		}
		p.pos++
		var inside, left, right imageMathBox
		inside, err = p.sequence(depth+1, closing)
		if err == nil {
			left, err = p.glyph(r, index)
		}
		if err == nil {
			right, err = p.glyph(closing, p.pos-1)
		}
		if err == nil {
			b = joinMathBoxes([]imageMathBox{left, inside, right}, p.gap(index))
		}
	default:
		p.pos++
		if strings.ContainsRune("_^{})）", r) || r == '\u0305' || r == '\u0304' {
			return b, fmt.Errorf("公式符号格式无效，请检查上下标和括号")
		}
		b, err = p.glyph(r, index)
	}
	if err != nil {
		return b, err
	}
	return p.scripts(b, depth)
}

func (p *formulaParser) scripts(base imageMathBox, depth int) (imageMathBox, error) {
	var sub, super *imageMathBox
	for p.pos < len(p.input) {
		r := p.input[p.pos]
		if r == '\u0305' || r == '\u0304' {
			p.pos++
			base = mathOverbar(base, p.font.Units)
			continue
		}
		if r != '_' && r != '^' {
			break
		}
		if (r == '_' && sub != nil) || (r == '^' && super != nil) {
			return base, fmt.Errorf("同一基项不能重复添加上下标，请合并内容")
		}
		p.pos++
		// Keep additional body spacing in millimetres when shrinking scripts.
		size := p.settings.FontSize
		if p.inline {
			p.settings.FontSize *= .65
		}
		script, err := p.group(depth)
		p.settings.FontSize = size
		if err != nil {
			return base, err
		}
		script = scaleFormulaBox(script, .65)
		if r == '_' {
			sub = &script
		} else {
			super = &script
		}
	}
	if sub == nil && super == nil {
		return base, nil
	}
	return p.attachScripts(base, sub, super), nil
}

func (p *formulaParser) axis() float64 {
	b, ok := (&imageMathParser{font: p.font}).glyph('=')
	if ok {
		return (b.top + b.bottom) / 2
	}
	return -.3 * p.font.Units
}
