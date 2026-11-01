package engine

// Tables and text boxes use the largest uniform size that fits their content,
// including merged spans, explicit lines and formulas.
func (f *Font) tableFontSize(t TableAsset) float64 {
	xs, ys := tableOffsets(t.ColumnWidths), tableOffsets(t.RowHeights)
	size, hasText := 40.0, false
	for _, c := range t.Cells {
		width, height := f.MeasureTableText(c.Text, 1)
		if width > 0 {
			size = min(size, (xs[c.Col+c.ColSpan]-xs[c.Col]-2*t.Padding)/width)
			hasText = true
		}
		if height > 0 {
			size = min(size, (ys[c.Row+c.RowSpan]-ys[c.Row]-2*t.Padding)/height)
			hasText = true
		}
	}
	if !hasText {
		for _, height := range t.RowHeights {
			size = min(size, height-2*t.Padding)
		}
	}
	return max(1, size)
}
