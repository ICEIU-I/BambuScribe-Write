package engine

// BlankCaretStop is a prospective insertion on an unused row. It does not add
// characters or ink; the editor inserts BlankLines only when typing begins.
type BlankCaretStop struct {
	CaretStop
	BlankLines int `json:"blankLines"`
}

func appendBlankCarets(page *Page, s LayoutSettings, objects []RegionBounds, row, index, padding int, y float64) {
	left := s.MarginLeft + s.Indent*(s.FontSize*s.LetterSpacing/100+s.LetterGap)
	right, bottom := s.PaperWidth-s.MarginRight, s.PaperHeight-s.MarginBottom
	for ; row < page.LineCount && y+min(s.FontSize, page.LineHeight) <= bottom+1e-8; row, padding, y = row+1, padding+1, y+page.LineHeight+s.ParagraphGap {
		spans := availableTextSpans(left, right, y, min(s.FontSize, page.LineHeight, bottom-y), objects)
		for _, span := range spans {
			// Leave room for a writing cell; fully covered rows still count as
			// newlines but must not advertise an insertion underneath an object.
			if span.right-span.left < s.FontSize*s.LetterSpacing/100-1e-8 {
				continue
			}
			x := span.left + lineOrigin(s, 0, span.right-span.left, 0) - s.MarginLeft
			page.BlankCarets = append(page.BlankCarets, BlankCaretStop{CaretStop{index, x, y}, padding})
			break
		}
	}
}
