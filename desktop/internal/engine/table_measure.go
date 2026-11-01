package engine

// MeasureTableText uses the same spacing, explicit line breaks, glyph bounds
// and subscripts as native table rendering. Importers must not guess font size.
func (f *Font) MeasureTableText(text string, size float64) (width, height float64) {
	t := f.tableText(text, size, 1e9)
	return tableTextDimensions(t)
}

// Imported captions include body spacing, including fixed millimeter gaps.
func (f *Font) MeasureTextBoxText(text string, size float64, settings LayoutSettings) (width, height float64) {
	settings = settings.normalized()
	return tableTextDimensions(f.tableTextWithSpacing(text, size, 1e9, &settings))
}

func tableTextDimensions(t tableText) (width, height float64) {
	for _, w := range t.widths {
		width = max(width, w)
	}
	return width, t.height
}
