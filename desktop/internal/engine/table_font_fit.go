package engine

import "math"

// Keep the smallest effective size as the table summary for older clients.
// Each cell fits independently so a long heading cannot shrink short values.
func (f *Font) tableFontSize(t TableAsset, settings LayoutSettings) float64 {
	size, hasText := tableFontLimit(t, settings), false
	for _, c := range t.Cells {
		if c.Text != "" {
			size = min(size, f.tableCellFontSize(t, c, settings))
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

func (f *Font) tableCellFontSize(t TableAsset, c TableCell, settings LayoutSettings) float64 {
	if t.isTextBox() {
		return min(tableFontLimit(t, settings), f.textBoxFontSize(t, c, settings))
	}
	return min(tableFontLimit(t, settings), f.unlimitedTableCellFont(t, c))
}

func (f *Font) unlimitedTableCellFont(t TableAsset, c TableCell) float64 {
	xs, ys := tableOffsets(t.ColumnWidths), tableOffsets(t.RowHeights)
	width, height := f.MeasureTableText(c.Text, 1)
	size := math.Inf(1)
	if width > 0 {
		size = min(size, (xs[c.Col+c.ColSpan]-xs[c.Col]-2*t.Padding)/width)
	}
	if height > 0 {
		size = min(size, (ys[c.Row+c.RowSpan]-ys[c.Row]-2*t.Padding)/height)
	}
	return max(1, size)
}
