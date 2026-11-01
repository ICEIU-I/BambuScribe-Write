package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"fmt"
)

// Keep the native table editor's minimum font size and physical cell padding.
const minImportFontSize = 1.0

func fitTableColumns(f *engine.Font, t Table, size, width, pad float64) (float64, []float64, error) {
	// Text widths scale linearly, including explicit newlines and subscripts.
	// Measure once so the search does not repeatedly traverse glyph strokes.
	widths := make([]float64, len(t.Cells))
	for i, c := range t.Cells {
		widths[i], _ = f.MeasureTableText(cellText(c.Text), 1)
	}
	columns := minimumColumns(t, widths, size, pad)
	if sum(columns) <= width {
		return size, columns, nil
	}
	columns = minimumColumns(t, widths, minImportFontSize, pad)
	if sum(columns) > width {
		return 0, nil, fmt.Errorf("表格缩小至最小字号 1 mm 后仍需 %.2f mm 宽，纸面可用 %.2f mm；请加宽纸张或减少内容", sum(columns), width)
	}
	low, high := minImportFontSize, size
	for range 40 {
		mid := (low + high) / 2
		candidate := minimumColumns(t, widths, mid, pad)
		if sum(candidate) <= width {
			low, columns = mid, candidate
		} else {
			high = mid
		}
	}
	return low, columns, nil
}

func minimumColumns(t Table, widths []float64, size, pad float64) []float64 {
	columns := make([]float64, len(t.Columns))
	for i := range columns {
		columns[i] = max(3, size*.65+2*pad)
	}
	// Solve ordinary cells before spans so each span adds only the deficit.
	for _, spanning := range []bool{false, true} {
		for i, c := range t.Cells {
			if (c.ColSpan > 1) != spanning {
				continue
			}
			required := widths[i]*size + 2*pad + .05
			used := sum(columns[c.Col : c.Col+c.ColSpan])
			if required > used {
				for j := c.Col; j < c.Col+c.ColSpan; j++ {
					columns[j] += (required - used) / float64(c.ColSpan)
				}
			}
		}
	}
	return columns
}

func fitTextBoxFont(f *engine.Font, text string, size, width float64) (float64, error) {
	w, _ := f.MeasureTableText(text, size)
	if w+1 <= width {
		return size, nil
	}
	minimum, _ := f.MeasureTableText(text, minImportFontSize)
	if minimum+1 > width {
		return 0, fmt.Errorf("表名或说明缩小至最小字号 1 mm 后仍需 %.2f mm 宽，纸面可用 %.2f mm；请加宽纸张或减少文字", minimum+1, width)
	}
	// Leave a tiny rounding margin for the native renderer's width check.
	return max(minImportFontSize, size*(width-1)/w-1e-9), nil
}
