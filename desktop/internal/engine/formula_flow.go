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
	appendGap := func(text string, x, atY, width, height float64) {
		if text == "" {
			return
		}
		page := &doc.Pages[len(doc.Pages)-1]
		start := *textIndex
		*textIndex += bodyLength([]rune(text))
		page.Text += text
		page.Cells = append(page.Cells, GlyphCell{start, *textIndex, x, atY, width, height})
		page.Carets = append(page.Carets, CaretStop{start, x, atY}, CaretStop{*textIndex, x + width, atY})
	}
	appendTrailing := func(span formulaSpan) {
		if span.trailing != "" {
			page := &doc.Pages[len(doc.Pages)-1]
			end := page.Carets[len(page.Carets)-1]
			appendGap(span.trailing, end.X, end.Y, 0, page.LineHeight)
		}
	}
	for at := 0; at < len(spans); {
		page := &doc.Pages[len(doc.Pages)-1]
		if *row >= page.LineCount || *y >= bottom-1e-8 {
			newPage()
			*y = s.MarginTop
			page = &doc.Pages[len(doc.Pages)-1]
		}
		pitch := page.LineHeight
		boxes := []imageMathBox{}
		used, index := 0.0, *textIndex
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
			gap := 0.0
			if next > at {
				gap = gapWidth
			}
			// A long summary equation gets its own row, even if a tiny item
			// would fit beside it. Other complete equations fill the row.
			if next > at && (used+gap+lines[0].box.width > width+1e-8 || lines[0].box.width > width*.5) {
				break
			}
			boxes = append(boxes, lines[0].box)
			used += gap + lines[0].box.width
			index += bodyLength([]rune(span.opening + span.source + span.closing))
			if used > width*.5 && len(boxes) == 1 {
				break
			}
		}
		if len(boxes) == 0 {
			span := spans[at]
			appendGap(span.gap, s.MarginLeft, *y, 0, pitch)
			if err := f.appendBodyFormula(doc, s, span.source, span.opening, span.closing, occupied, ink, y, row, textIndex, newPage, s.variation(fmt.Sprintf("formula:%d", textOffset+*textIndex))); err != nil {
				return err
			}
			appendTrailing(span)
			at++
			continue
		}
		clear := *y+pitch <= bottom+1e-8 && len(availableTextSpans(s.MarginLeft, s.MarginLeft+width, *y, pitch, occupied[len(doc.Pages)-1])) > 0
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
			span := spans[at+i]
			gap := 0.0
			if i > 0 {
				gap = gapWidth
			}
			appendGap(span.gap, x, *y, gap, pitch)
			x += gap
			f.appendPlacedFormula(page, box, span.source, span.opening+span.source+span.closing, x, *y, pitch, textIndex)
			appendTrailing(span)
			x += box.width
		}
		at += len(boxes)
		*y += pitch
		*row++
		*ink = textInk{}
	}
	return nil
}
