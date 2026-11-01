package engine

import (
	"fmt"
	"math"
	"strconv"
)

// appendBodyFormula shares body pagination and obstacle avoidance while reserving
// enough height for scaled numerators, denominators and all equality terms.
func (f *Font) appendBodyFormula(doc *Document, s LayoutSettings, source, opening, closing string, occupied map[int][]RegionBounds, ink *textInk, y *float64, row *int, textIndex *int, newPage func(), variation handwritingVariation) error {
	bottom := s.PaperHeight - s.MarginBottom
	width := s.PaperWidth - s.MarginLeft - s.MarginRight
	lines, err := f.formulaLines(source, s, (s.PaperWidth - s.MarginLeft - s.MarginRight), variation)
	if err != nil {
		return err
	}
	// Reserve the complete equation before emitting any source text.
	// This keeps formula tokens intact when copying/reordering pages.
	for _, old := range ink.active {
		*y = max(*y, old.Y+old.Height+s.FontSize*.1)
	}
	for {
		if *row >= doc.Pages[len(doc.Pages)-1].LineCount || *y >= bottom-1e-8 {
			newPage()
			*y = s.MarginTop
		}
		pitch := doc.Pages[len(doc.Pages)-1].LineHeight
		total := 0.0
		for i, line := range lines {
			h := line.box.bottom - line.box.top
			if i == len(lines)-1 {
				total += h
			} else {
				total += h + variation.gap(i, s.FontSize*.35)
			}
		}
		if total > bottom-s.MarginTop {
			return fmt.Errorf("计算式超过一页高度，请拆分计算步骤")
		}
		if *y+total > bottom {
			newPage()
			*y = s.MarginTop
			continue
		}
		if len(availableTextSpans(s.MarginLeft, s.MarginLeft+width, *y, total, occupied[len(doc.Pages)-1])) == 0 {
			*y += pitch
			*row++
			continue
		}
		break
	}
	for li, line := range lines {
		b := line.box
		height := b.bottom - b.top
		page := &doc.Pages[len(doc.Pages)-1]
		x := lineOrigin(s, 0, width, b.width)
		sourceText := line.source
		if li == 0 {
			sourceText = opening + sourceText
		}
		if li == len(lines)-1 {
			sourceText += closing
		}
		start := *textIndex
		for _, r := range sourceText {
			*textIndex += utf16Width(r)
		}
		strokes := appendMathStrokes(nil, b.strokes, x, *y-b.top, 1)
		page.Strokes = append(page.Strokes, strokes...)
		page.Ink = append(page.Ink, InkLayer{"", strconv.Itoa(start), inkSource([]string{f.ID, sourceText}), strokes})
		page.Cells = append(page.Cells, GlyphCell{start, *textIndex, x, *y, b.width, height})
		page.Carets = append(page.Carets, CaretStop{start, x, *y}, CaretStop{*textIndex, x + b.width, *y})
		page.Formulas = append(page.Formulas, FormulaPlacement{Start: start, End: *textIndex, Source: line.source, RegionBounds: RegionBounds{x, *y, b.width, height}})
		page.Text += sourceText
		*y += height + variation.gap(li, s.FontSize*.35)
		*row = int(math.Floor((*y - s.MarginTop) / page.LineHeight))
		*ink = textInk{}
	}

	return nil
}
