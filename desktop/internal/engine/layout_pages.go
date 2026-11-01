package engine

func blankLayoutPage(s LayoutSettings, index int) Page {
	count, height := s.rows(index)
	return Page{
		Number: index + 1, LineCount: count, LineHeight: height,
		Strokes: []Stroke{}, TextStrokes: []Stroke{}, Missing: []MissingGlyph{},
		Cells: []GlyphCell{}, Carets: []CaretStop{},
	}
}
