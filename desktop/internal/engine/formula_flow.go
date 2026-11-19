package engine

import (
	"fmt"
	"unicode"
)

type formulaSpan struct {
	gap, opening, source, closing, trailing string
}

// A paragraph can contain several complete equations separated by whitespace.
// Match each delimiter pair, rather than stripping the first and last pair.
func formulaParagraph(text string) ([]formulaSpan, bool) {
	runes := []rune(text)
	var spans []formulaSpan
	for i := 0; i < len(runes); {
		gapStart := i
		for i < len(runes) && unicode.IsSpace(runes[i]) {
			i++
		}
		if i == len(runes) && len(spans) > 0 {
			spans[len(spans)-1].trailing = string(runes[gapStart:])
			break
		}
		if i+1 >= len(runes) || runes[i] != '\\' || (runes[i+1] != '(' && runes[i+1] != '[') {
			return nil, false
		}
		start := i
		stack := []rune{}
		end := 0
		for ; i+1 < len(runes); i++ {
			if runes[i] != '\\' {
				continue
			}
			i++
			switch runes[i] {
			case '(':
				stack = append(stack, ')')
			case '[':
				stack = append(stack, ']')
			case ')', ']':
				if len(stack) == 0 || stack[len(stack)-1] != runes[i] {
					return nil, false
				}
				stack = stack[:len(stack)-1]
				if len(stack) == 0 {
					end = i + 1
				}
			}
			if end > 0 {
				break
			}
		}
		if end == 0 {
			return nil, false
		}
		spans = append(spans, formulaSpan{gap: string(runes[gapStart:start]), opening: string(runes[start : start+2]), source: string(runes[start+2 : end-2]), closing: string(runes[end-2 : end])})
		i = end
	}
	return spans, len(spans) > 0
}

// Lay out complete short equations side by side. Long equations retain the
// existing equality-term wrapping. Each equation keeps its source and hit box.
func (f *Font) appendFormulaFlow(doc *Document, s LayoutSettings, spans []formulaSpan, occupied map[int][]RegionBounds, ink *textInk, y *float64, row, textIndex *int, textOffset int, newPage func()) error {
	width := s.PaperWidth - s.MarginLeft - s.MarginRight
	bottom := s.PaperHeight - s.MarginBottom
	gapWidth := max(4, s.FontSize*.75)
	for at := 0; at < len(spans); {
		page := &doc.Pages[len(doc.Pages)-1]
		if *row >= page.LineCount || *y >= bottom-1e-8 {
			newPage()
			*y = s.MarginTop
			page = &doc.Pages[len(doc.Pages)-1]
		}
		pitch := page.LineHeight
		boxes := []imageMathBox{}
		boxRows := []int{}
		gaps := []float64{}
		used, index := 0.0, *textIndex
		rows := 1
		for next := at; next < len(spans); next++ {
			span := spans[next]
			index += bodyLength([]rune(span.gap))
			lines, err := f.formulaRowLines(span.source, s, width, pitch, s.variation(fmt.Sprintf("formula:%d", textOffset+index)))
			if err != nil {
				return err
			}
			if len(lines) != 1 {
				break
			}
			gap := f.formulaWhitespaceWidth(span.gap, s)
			if next > at {
				gap = max(gapWidth, gap)
			}
			// Pack using the readable size, without reducing letters to make
			// another equation fit beside the current one.
			if used+gap+lines[0].box.width > width+1e-8 {
				break
			}
			rows = max(rows, lines[0].rows)
			boxRows = append(boxRows, lines[0].rows)
			boxes = append(boxes, lines[0].box)
			gaps = append(gaps, gap)
			used += gap + lines[0].box.width
			index += bodyLength([]rune(span.opening + span.source + span.closing))
		}
		if len(boxes) == 0 {
			span := spans[at]
			index := *textIndex + bodyLength([]rune(span.gap))
			if err := f.appendBodyFormula(doc, s, span, occupied, ink, y, row, textIndex, newPage, s.variation(fmt.Sprintf("formula:%d", textOffset+index))); err != nil {
				return err
			}
			at++
			continue
		}
		height := float64(rows) * pitch
		if rows > page.LineCount || height > bottom-s.MarginTop+1e-8 {
			return fmt.Errorf("大分式需要两行空间，请增加每页行数或拆分计算式")
		}
		clear := *row+rows <= page.LineCount && *y+height <= bottom+1e-8 && len(availableTextSpans(s.MarginLeft, s.MarginLeft+width, *y, height, occupied[len(doc.Pages)-1])) > 0
		for _, b := range boxes {
			for _, old := range ink.active {
				clear = clear && old.Y+old.Height+bodyInkWidth/2 <= *y+b.top+1e-8
			}
		}
		if !clear {
			*y += pitch
			*row++
			continue
		}
		x := lineOrigin(s, 0, width, used)
		for i, box := range boxes {
			// Centre a short result beside a taller fraction without reserving
			// a separate mostly-empty row for the result.
			shift := float64(rows-boxRows[i]) * pitch / 2
			box.strokes = appendMathStrokes(nil, box.strokes, 0, shift, 1)
			box.top, box.bottom = box.top+shift, box.bottom+shift
			span := spans[at+i]
			f.appendFormulaWhitespace(page, span.gap, s, x, *y, gaps[i], textIndex)
			x += gaps[i]
			f.appendPlacedFormula(page, box, span.source, span.opening+span.source+span.closing, x, *y, height, textIndex)
			f.appendFormulaTrailing(page, span.trailing, s, textIndex)
			x += box.width
		}
		at += len(boxes)
		*y += height
		*row += rows
		*ink = textInk{}
	}
	return nil
}
