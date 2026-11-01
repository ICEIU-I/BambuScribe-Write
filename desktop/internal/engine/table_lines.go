package engine

// Atomic grid edges are shared by both adjacent cells. Explicit hiding wins,
// and contiguous segments join into one stroke so the pen never redraws a seam.
func tableLines(t TableAsset, rows []float64) []Stroke {
	nr, nc := len(rows), len(t.ColumnWidths)
	owners := make([][]int, nr)
	for r := range owners {
		owners[r] = make([]int, nc)
	}
	for i, c := range t.Cells {
		for r := c.Row; r < c.Row+c.RowSpan; r++ {
			for col := c.Col; col < c.Col+c.ColSpan; col++ {
				owners[r][col] = i
			}
		}
	}
	h := make([][]bool, nr+1)
	v := make([][]bool, nc+1)
	for r := range h {
		h[r] = make([]bool, nc)
		for c := range h[r] {
			boundary := r == 0 || r == nr || owners[r-1][c] != owners[r][c]
			h[r][c] = boundary && (t.Style == "grid" || t.Style == "horizontal" || (t.Style == "three-line" && (r == 0 || r == 1 || r == nr)))
		}
	}
	for c := range v {
		v[c] = make([]bool, nr)
		for r := range v[c] {
			v[c][r] = t.Style == "grid" && (c == 0 || c == nc || owners[r][c-1] != owners[r][c])
		}
	}
	// Two passes make false overrides deterministic at shared edges.
	for _, enabled := range []bool{true, false} {
		for _, c := range t.Cells {
			b := c.Borders
			if b == nil {
				continue
			}
			set := func(value *bool, edge []bool, start, end int) {
				if value != nil && *value == enabled {
					for i := start; i < end; i++ {
						edge[i] = enabled
					}
				}
			}
			set(b.Top, h[c.Row], c.Col, c.Col+c.ColSpan)
			set(b.Bottom, h[c.Row+c.RowSpan], c.Col, c.Col+c.ColSpan)
			set(b.Left, v[c.Col], c.Row, c.Row+c.RowSpan)
			set(b.Right, v[c.Col+c.ColSpan], c.Row, c.Row+c.RowSpan)
		}
	}
	xs, ys := tableOffsets(t.ColumnWidths), tableOffsets(rows)
	out := []Stroke{}
	for r, edges := range h {
		for c := 0; c < nc; {
			if !edges[c] {
				c++
				continue
			}
			start := c
			for c < nc && edges[c] {
				c++
			}
			out = append(out, Stroke{{xs[start], ys[r]}, {xs[c], ys[r]}})
		}
	}
	for c, edges := range v {
		for r := 0; r < nr; {
			if !edges[r] {
				r++
				continue
			}
			start := r
			for r < nr && edges[r] {
				r++
			}
			out = append(out, Stroke{{xs[c], ys[start]}, {xs[c], ys[r]}})
		}
	}
	return out
}
func tableOffsets(sizes []float64) []float64 {
	out := make([]float64, len(sizes)+1)
	for i, v := range sizes {
		out[i+1] = out[i] + v
	}
	return out
}
