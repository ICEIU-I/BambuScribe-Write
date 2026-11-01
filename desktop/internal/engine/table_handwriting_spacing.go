package engine

import (
	"fmt"
	"strings"
)

// Borderless imported equations use two-row fractions separated by blank rows.
// Translate each whole equation inside the blank separators, keeping fractions
// together and the object's saved geometry unchanged.
func varyTableBands(t TableAsset, out *TablePlacement, settings LayoutSettings) {
	if settings.HandwritingVariation <= 0 || t.Style != "none" || len(out.RowHeights) < 3 {
		return
	}
	ys := tableOffsets(out.RowHeights)
	occupied := make([]bool, len(out.RowHeights))
	for _, c := range t.Cells {
		if strings.TrimSpace(c.Text) != "" {
			for r := c.Row; r < c.Row+c.RowSpan; r++ {
				occupied[r] = true
			}
		}
	}
	rules := len(tableLines(t, out.RowHeights))
	// A ruled empty row is not a separator. Keep connected table grids fixed.
	for _, s := range out.Strokes[:rules] {
		if len(s) < 2 {
			continue
		}
		low, high := s[0].Y, s[len(s)-1].Y
		if low > high {
			low, high = high, low
		}
		for r := range occupied {
			if low < ys[r+1]-1e-8 && high > ys[r]+1e-8 {
				occupied[r] = true
			}
		}
	}
	offsets := make([]float64, len(occupied))
	v := settings.variation(fmt.Sprintf("table-bands:%s", t.ID))
	for start := 0; start < len(occupied); {
		if !occupied[start] {
			start++
			continue
		}
		end := start + 1
		for end < len(occupied) && occupied[end] {
			end++
		}
		before, after := 0.0, 0.0
		if start > 0 {
			before = out.RowHeights[start-1] / 2
		}
		if end < len(occupied) {
			after = out.RowHeights[end] / 2
		}
		dy := v.spaceOffset(start, true, max(t.FontSize, settings.FontSize), before, after)
		for r := start; r < end; r++ {
			offsets[r] = dy
		}
		start = end
	}
	for i := range out.Cells {
		c := &out.Cells[i]
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		dy := offsets[c.Row]
		for _, stroke := range c.Strokes {
			for j := range stroke {
				stroke[j].Y += dy
			}
		}
		for j := range c.Missing {
			c.Missing[j].Y += dy
		}
	}
	// Cell strokes share their slices with the combined table strokes.
	for _, stroke := range out.Strokes[:rules] {
		if len(stroke) < 2 {
			continue
		}
		y := stroke[0].Y
		if stroke[len(stroke)-1].Y != y {
			continue
		}
		for r := range offsets {
			if y >= ys[r]-1e-8 && y <= ys[r+1]+1e-8 && occupied[r] {
				for j := range stroke {
					stroke[j].Y += offsets[r]
				}
				break
			}
		}
	}
}
