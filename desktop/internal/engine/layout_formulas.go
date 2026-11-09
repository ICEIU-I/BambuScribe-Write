package engine

import (
	"fmt"
	"strconv"
)

// Display formulas share the ruled body grid. Fit each complete rendered line
// uniformly and reserve the whole equation when avoiding objects or changing pages.
func (f *Font) appendBodyFormula(doc *Document, s LayoutSettings, source, opening, closing string, occupied map[int][]RegionBounds, ink *textInk, y *float64, row *int, textIndex *int, newPage func(), variation handwritingVariation) error {
	bottom := s.PaperHeight - s.MarginBottom
	width := s.PaperWidth - s.MarginLeft - s.MarginRight
	var lines []formulaLine
	for {
		if *row >= doc.Pages[len(doc.Pages)-1].LineCount || *y >= bottom-1e-8 {
			newPage()
			*y = s.MarginTop
		}
		pitch := doc.Pages[len(doc.Pages)-1].LineHeight
		var err error
		lines, err = f.formulaRowLines(source, s, width, pitch, variation)
		if err != nil {
			return err
		}
		total := float64(len(lines)) * pitch
		if total > bottom-s.MarginTop+1e-8 {
			return fmt.Errorf("计算式超过一页高度，请拆分计算步骤")
		}
		if *y+total > bottom+1e-8 {
			newPage()
			*y = s.MarginTop
			continue
		}
		clear := true
		for _, old := range ink.active {
			clear = clear && old.Y+old.Height+bodyInkWidth/2 <= *y+lines[0].box.top+1e-8
		}
		if !clear || len(availableTextSpans(s.MarginLeft, s.MarginLeft+width, *y, total, occupied[len(doc.Pages)-1])) == 0 {
			*y += pitch
			*row++
			continue
		}
		break
	}
	for li, line := range lines {
		b := line.box
		page := &doc.Pages[len(doc.Pages)-1]
		height := page.LineHeight
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
		strokes := appendMathStrokes(nil, b.strokes, x, *y, 1)
		page.Strokes = append(page.Strokes, strokes...)
		page.Ink = append(page.Ink, InkLayer{"", strconv.Itoa(start), inkSource([]string{f.ID, sourceText}), strokes})
		page.Cells = append(page.Cells, GlyphCell{start, *textIndex, x, *y, b.width, height})
		page.Carets = append(page.Carets, CaretStop{start, x, *y}, CaretStop{*textIndex, x + b.width, *y})
		page.Formulas = append(page.Formulas, FormulaPlacement{Start: start, End: *textIndex, Source: line.source, RegionBounds: RegionBounds{x, *y + b.top, b.width, b.bottom - b.top}})
		page.Text += sourceText
		*y += height
		*row++
		*ink = textInk{}
	}

	return nil
}
