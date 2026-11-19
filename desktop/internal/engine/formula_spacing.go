package engine

func (f *Font) formulaWhitespaceWidth(text string, s LayoutSettings) float64 {
	width := 0.0
	for _, r := range text {
		width += f.advance(r, s)
	}
	return width
}

// Whitespace outside delimiters is editable body spacing. Keep a caret for
// every character; any minimum equation separator is added to its last cell.
func (f *Font) appendFormulaWhitespace(page *Page, text string, s LayoutSettings, x, y, width float64, textIndex *int) {
	endX := min(x+width, s.PaperWidth-s.MarginRight)
	runes := []rune(text)
	for i, r := range runes {
		start := *textIndex
		*textIndex += utf16Width(r)
		nextX := min(x+f.advance(r, s), endX)
		if i == len(runes)-1 {
			nextX = endX
		}
		page.Cells = append(page.Cells, GlyphCell{start, *textIndex, x, y, nextX - x, page.LineHeight})
		page.Carets = append(page.Carets, CaretStop{start, x, y}, CaretStop{*textIndex, nextX, y})
		x = nextX
	}
	page.Text += text
}

func (f *Font) appendFormulaTrailing(page *Page, text string, s LayoutSettings, textIndex *int) {
	if text == "" {
		return
	}
	end := page.Carets[len(page.Carets)-1]
	f.appendFormulaWhitespace(page, text, s, end.X, end.Y, f.formulaWhitespaceWidth(text, s), textIndex)
}
