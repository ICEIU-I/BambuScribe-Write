package engine

import (
	"fmt"
	"strings"
)

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
	b, err := p.sequence(0, 0)
	if err == nil && p.pos != len(p.input) {
		err = fmt.Errorf("公式括号不匹配")
	}
	return scaleFormulaBox(b, s.FontSize/f.Units), err
}
func formulaSource(paragraph string) (string, bool) {
	if (strings.HasPrefix(paragraph, `\(`) && strings.HasSuffix(paragraph, `\)`)) ||
		(strings.HasPrefix(paragraph, `\[`) && strings.HasSuffix(paragraph, `\]`)) {
		return paragraph[2 : len(paragraph)-2], true
	}
	return "", false
}

// Split only between complete equality terms. Fractions never straddle lines.
func (f *Font) formulaLines(source string, s LayoutSettings, width float64, variation handwritingVariation) ([]formulaLine, error) {
	runes := []rune(source)
	starts := []int{0}
	depth := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' && i+1 < len(runes) && strings.ContainsRune(`_^{}\`, runes[i+1]) {
			i++
			continue
		}
		if strings.ContainsRune("{(（[", r) {
			depth++
		}
		if strings.ContainsRune("})）]", r) {
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
