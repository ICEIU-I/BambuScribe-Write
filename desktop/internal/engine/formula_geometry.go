package engine

func joinMathBoxes(boxes []imageMathBox, gap float64) imageMathBox {
	out := boxes[0]
	for _, b := range boxes[1:] {
		out.strokes = appendMathStrokes(out.strokes, b.strokes, out.width+gap, 0, 1)
		out.width += gap + b.width
		out.top, out.bottom = min(out.top, b.top), max(out.bottom, b.bottom)
	}
	return out
}

func mathOverbar(b imageMathBox, unit float64) imageMathBox {
	b.top -= unit * .10
	b.strokes = append(b.strokes, Stroke{{0, b.top}, {b.width, b.top}})
	return b
}

func (p *formulaParser) fraction(n, d imageMathBox) imageMathBox {
	n, d = scaleFormulaBox(n, p.ratio), scaleFormulaBox(d, p.ratio)
	unit, axis := p.font.Units, p.axis()
	pad, space := unit*.08, unit*.10
	b := imageMathBox{width: max(n.width, d.width) + 2*pad}
	ny, dy := axis-space-n.bottom, axis+space-d.top
	b.top, b.bottom = n.top+ny, d.bottom+dy
	b.strokes = appendMathStrokes(b.strokes, n.strokes, (b.width-n.width)/2, ny, 1)
	b.strokes = appendMathStrokes(b.strokes, d.strokes, (b.width-d.width)/2, dy, 1)
	b.strokes = append(b.strokes, Stroke{{0, axis}, {b.width, axis}})
	return b
}

func (p *formulaParser) radical(b imageMathBox, degree *imageMathBox) imageMathBox {
	unit := p.font.Units
	stem, gap := unit*.45, unit*.08
	if p.inline {
		stem = max(unit*.5, unit*.5*p.settings.LetterSpacing/100)
		gap = p.gap(0)
	}
	left := 0.0
	if degree != nil {
		d := scaleFormulaBox(*degree, .5)
		degree = &d
		left = max(0, d.width-stem*.4)
	}
	bar := b.top - unit*.14
	h := b.bottom - b.top
	rules := []Stroke{
		{{left, b.top + h*.6}, {left + stem*.22, b.top + h*.5}, {left + stem*.45, b.bottom}, {left + stem*.85, bar}},
		{{left + stem*.85, bar}, {left + stem + gap + b.width, bar}},
	}
	b.strokes = appendMathStrokes(rules, b.strokes, left+stem+gap, 0, 1)
	b.width += left + stem + gap
	b.top = bar
	if degree != nil {
		dy := bar + unit*.16 - degree.bottom
		b.strokes = appendMathStrokes(b.strokes, degree.strokes, 0, dy, 1)
		b.top = min(b.top, degree.top+dy)
	}
	return b
}

func (p *formulaParser) attachScripts(b imageMathBox, sub, super *imageMathBox) imageMathBox {
	unit := p.font.Units
	gap := unit * .025
	subY := unit * .12
	if p.inline {
		gap = p.gap(0)
		subY = unit * .05 // .65*glyph + .4 em in body coordinates.
	}
	x, width := b.width+gap, 0.0
	if super != nil {
		dy := b.top - super.bottom + unit*.15
		if sub != nil {
			dy = min(dy, sub.top+subY-super.bottom-unit*.10)
		}
		b.strokes = appendMathStrokes(b.strokes, super.strokes, x, dy, 1)
		b.top, b.bottom = min(b.top, super.top+dy), max(b.bottom, super.bottom+dy)
		width = super.width
	}
	if sub != nil {
		b.strokes = appendMathStrokes(b.strokes, sub.strokes, x, subY, 1)
		b.top, b.bottom = min(b.top, sub.top+subY), max(b.bottom, sub.bottom+subY)
		width = max(width, sub.width)
	}
	b.width = x + width
	return b
}
