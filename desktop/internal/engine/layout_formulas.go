package engine

import (
	"fmt"
	"strconv"
)

// Display formulas share the ruled body grid. Fit each complete rendered line
// uniformly and reserve the whole equation when avoiding objects or changing pages.
func (f *Font) appendBodyFormula(doc *Document, s LayoutSettings, span formulaSpan, occupied map[int][]RegionBounds, ink *textInk, y *float64, row *int, textIndex *int, newPage func(), variation handwritingVariation) error {
	bottom := s.PaperHeight - s.MarginBottom
	width := s.PaperWidth - s.MarginLeft - s.MarginRight
	indent := f.formulaWhitespaceWidth(span.gap, s)
	if indent >= width {
		return fmt.Errorf("公式缩进过大，请减少前面的空格或 Tab")
	}
	var lines []formulaLine
	for {
		if *row >= doc.Pages[len(doc.Pages)-1].LineCount || *y >= bottom-1e-8 {
			newPage()
			*y = s.MarginTop
		}
		pitch := doc.Pages[len(doc.Pages)-1].LineHeight
		var err error
		lines, err = f.formulaRowLines(span.source, s, width-indent, pitch, variation)
		if err != nil {
			if indent > 0 {
				if _, fullErr := f.formulaRowLines(span.source, s, width, pitch, variation); fullErr == nil {
					return fmt.Errorf("公式缩进后宽度不足，请减少前面的空格或 Tab")
				}
			}
			return err
		}
		totalRows := 0
		for _, line := range lines {
			totalRows += line.rows
		}
		total := float64(totalRows) * pitch
		if total > bottom-s.MarginTop+1e-8 || totalRows > doc.Pages[len(doc.Pages)-1].LineCount {
			return fmt.Errorf("计算式超过一页高度，请拆分计算步骤")
		}
		if *y+total > bottom+1e-8 || *row+totalRows > doc.Pages[len(doc.Pages)-1].LineCount {
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
		height := float64(line.rows) * page.LineHeight
		x := lineOrigin(s, indent, width-indent, b.width)
		sourceText := line.source
		if li == 0 {
			f.appendFormulaWhitespace(page, span.gap, s, x-indent, *y, indent, textIndex)
			sourceText = span.opening + sourceText
		}
		if li == len(lines)-1 {
			sourceText += span.closing
		}
		f.appendPlacedFormula(page, b, line.source, sourceText, x, *y, height, textIndex)
		if li == len(lines)-1 {
			f.appendFormulaTrailing(page, span.trailing, s, textIndex)
		}
		*y += height
		*row += line.rows
		*ink = textInk{}
	}

	return nil
}

func (f *Font) appendPlacedFormula(page *Page, b imageMathBox, source, sourceText string, x, y, height float64, textIndex *int) {
	start := *textIndex
	for _, r := range sourceText {
		*textIndex += utf16Width(r)
	}
	strokes := appendMathStrokes(nil, b.strokes, x, y, 1)
	page.Strokes = append(page.Strokes, strokes...)
	page.Ink = append(page.Ink, InkLayer{"", strconv.Itoa(start), inkSource([]string{f.ID, sourceText}), strokes})
	page.Cells = append(page.Cells, GlyphCell{start, *textIndex, x, y, b.width, height})
	page.Carets = append(page.Carets, CaretStop{start, x, y}, CaretStop{*textIndex, x + b.width, y})
	page.Formulas = append(page.Formulas, FormulaPlacement{Start: start, End: *textIndex, Source: source, RegionBounds: RegionBounds{x, y + b.top, b.width, b.bottom - b.top}})
	page.Text += sourceText
}
