package engine

import (
	"math"
	"strings"
)

// Contract an oversized grid before capping its letters at the body size.
// Preserve proportions, readable labels, writable blanks and editor minimums.
func (f *Font) CompactTable(t TableAsset, settings LayoutSettings) TableAsset {
	if t.isTextBox() {
		return t
	}
	body := max(1, settings.normalized().FontSize)
	smallest, largest := math.Inf(1), 0.0
	for _, c := range t.Cells {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		size := f.unlimitedTableCellFont(t, c)
		smallest, largest = min(smallest, size), max(largest, size)
	}
	if largest <= body {
		return t
	}
	scale := min(1, max(body/largest, body*.7/smallest))
	for _, axis := range [][]float64{t.ColumnWidths, t.RowHeights} {
		for _, v := range axis {
			scale = min(1, max(scale, 3/v))
		}
	}
	if scale >= 1-1e-8 {
		return t
	}
	t.ColumnWidths = append([]float64(nil), t.ColumnWidths...)
	t.RowHeights = append([]float64(nil), t.RowHeights...)
	for i := range t.ColumnWidths {
		t.ColumnWidths[i] *= scale
	}
	for i := range t.RowHeights {
		t.RowHeights[i] *= scale
	}
	t.Padding *= scale
	return t
}

func tableFontLimit(t TableAsset, settings LayoutSettings) float64 {
	if t.isTextBox() && t.GroupID == "" {
		return 40
	}
	return max(1, settings.normalized().FontSize)
}
