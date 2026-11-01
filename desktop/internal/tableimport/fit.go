package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"fmt"
)

const minImportFontRatio = .7

func importMinimumSize(size float64) float64 { return max(1, size*minImportFontRatio) }

func fitTableColumns(f *engine.Font, t Table, size, width, pad float64) (Table, float64, []float64, error) {
	// Text widths scale linearly, including explicit newlines and subscripts.
	// Measure once so the search does not repeatedly traverse glyph strokes.
	t.Cells = append([]Cell(nil), t.Cells...)
	widths := make([]float64, len(t.Cells))
	for i := range t.Cells {
		t.Cells[i].Text = cellText(t.Cells[i].Text)
		widths[i], _ = f.MeasureTableText(t.Cells[i].Text, 1)
	}
	columns := minimumColumns(t, widths, size, pad)
	if sum(columns) <= width {
		return t, size, columns, nil
	}
	minimum := importMinimumSize(size)
	columns = minimumColumns(t, widths, minimum, pad)
	if sum(columns) > width {
		return wrapTableColumns(f, t, minimum, width, pad)
	}
	low, high := minimum, size
	for range 40 {
		mid := (low + high) / 2
		candidate := minimumColumns(t, widths, mid, pad)
		if sum(candidate) <= width {
			low, columns = mid, candidate
		} else {
			high = mid
		}
	}
	return t, low, columns, nil
}

func wrapTableColumns(f *engine.Font, t Table, size, width, pad float64) (Table, float64, []float64, error) {
	widths := make([]float64, len(t.Cells))
	for i, c := range t.Cells {
		for _, token := range wrapTokens(c.Text) {
			w, _ := f.MeasureTableText(token, 1)
			widths[i] = max(widths[i], w)
		}
	}
	columns := minimumColumns(t, widths, size, pad)
	if sum(columns) > width+1e-8 {
		return t, 0, nil, fmt.Errorf("按正文70%%（%.2f mm）排版仍需 %.2f mm 宽；请加宽纸张或缩短不可换行的公式", size, sum(columns))
	}
	distributeColumns(columns, t.Columns, width)
	for i, c := range t.Cells {
		text, err := wrapText(c.Text, sum(columns[c.Col:c.Col+c.ColSpan])-2*pad, func(text string) float64 {
			w, _ := f.MeasureTableText(text, size)
			return w
		})
		if err != nil {
			return t, 0, nil, err
		}
		t.Cells[i].Text = text
	}
	return t, size, columns, nil
}

func distributeColumns(columns, ratios []float64, width float64) {
	extra, total := max(0, width-sum(columns)), sum(ratios)
	for i, r := range ratios {
		columns[i] += extra * r / total
	}
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
