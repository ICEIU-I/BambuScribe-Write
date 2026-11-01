package engine

// MeasureTableText uses the same spacing, explicit line breaks, glyph bounds
// and subscripts as native table rendering. Importers must not guess font size.
func (f *Font) MeasureTableText(text string, size float64) (width, height float64) {
	t := f.tableText(text, size, 1e9)
	for _, w := range t.widths {
		width = max(width, w)
	}
	return width, t.height
}
