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
	spans, ok := formulaParagraph(paragraph)
	if ok && len(spans) == 1 && spans[0].gap == "" && spans[0].trailing == "" {
		return spans[0].source, true
	}
	return "", false
}

// Split only between complete equality terms. Fractions never straddle lines.
func (f *Font) formulaLines(source string, s LayoutSettings, width float64, variation handwritingVariation) ([]formulaLine, error) {
	return f.splitFormulaLines(source, s, width, variation, func(b imageMathBox) formulaLine { return formulaLine{box: b, rows: 1} })
}

func (f *Font) formulaRowLines(source string, s LayoutSettings, width, pitch float64, variation handwritingVariation) ([]formulaLine, error) {
	return f.splitFormulaLines(source, s, width, variation, func(b imageMathBox) formulaLine {
		pad := formulaRowPadding(pitch)
		height := b.bottom - b.top
		// Keep compact expressions on one rule. Tall fractions use two rules
		// before any scaling or width measurement, so they cannot be packed
		// into a row merely because their letters have become tiny.
		rows := 1
		if height > pitch-2*pad+1e-8 {
			rows = 2
		}
		available := float64(rows) * pitch
		b = scaleFormulaBox(b, min(1, (available-2*pad)/height))
		// Preserve the ordinary baseline when it fits; taller structures share
		// one uniform transform and the available space between the rules.
		shift := max(pad, min(s.FontSize+b.top, available-pad-(b.bottom-b.top))) - b.top
		b.strokes = appendMathStrokes(nil, b.strokes, 0, shift, 1)
		b.top, b.bottom = b.top+shift, b.bottom+shift
		return formulaLine{box: b, rows: rows, minWidthScale: .85}
	})
}

func formulaRowPadding(pitch float64) float64 {
	return min(pitch*.1, max(bodyInkWidth, pitch*.025))
}

func (f *Font) splitFormulaLines(source string, s LayoutSettings, width float64, variation handwritingVariation, fit func(imageMathBox) formulaLine) ([]formulaLine, error) {
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
	var line formulaLine
	for i := 0; i < len(starts)-1; i++ {
		next := starts[i+1]
		b, err := f.formulaBox(string(runes[start:next]), s, variation.at(start))
		if err != nil {
			return nil, err
		}
		candidate := fit(b)
		if candidate.box.width > width+1e-8 && end > start {
			line.source = string(runes[start:end])
			out = append(out, line)
			start = end
			b, err = f.formulaBox(string(runes[start:next]), s, variation.at(start))
			if err != nil {
				return nil, err
			}
			candidate = fit(b)
		}
		if candidate.box.width > width+1e-8 {
			// Equality wrapping has already been exhausted. Allow only a small
			// final width adjustment to an indivisible two-row term; never
			// shrink an entire calculation chain just to keep it on one line.
			scale := width / candidate.box.width
			if candidate.rows != 2 || candidate.minWidthScale == 0 || scale < candidate.minWidthScale {
				return nil, fmt.Errorf("单个分式过宽，请拆分计算步骤或加宽纸张")
			}
			top := candidate.box.top
			candidate.box = scaleFormulaBox(candidate.box, scale)
			shift := top - candidate.box.top
			candidate.box.strokes = appendMathStrokes(nil, candidate.box.strokes, 0, shift, 1)
			candidate.box.top, candidate.box.bottom = top, candidate.box.bottom+shift
		}
		line = candidate
		end = next
	}
	line.source = string(runes[start:end])
	out = append(out, line)
	return out, nil
}

type formulaLine struct {
	source        string
	box           imageMathBox
	rows          int
	minWidthScale float64
}
