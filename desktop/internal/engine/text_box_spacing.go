package engine

// Separate fixed millimeter gaps from scalable ink for the live resize preview.
type TextLineSpacing struct {
	Width float64 `json:"width"`
	Gap   float64 `json:"gap"`
}

type TextSpacingMetrics struct {
	Lines      []TextLineSpacing `json:"lines"`
	StrokeGaps [][2]float64      `json:"strokeGaps"`
}

func textLineGap(line []tableLetter, gap float64) float64 {
	units := -1
	for _, letter := range line {
		units++
		if letter.char == '\t' {
			units += tabCells - 1
		}
	}
	return float64(max(0, units)) * gap
}

func (f *Font) textBoxSpacing(plan tableText, gap float64, align string) *TextSpacingMetrics {
	out := &TextSpacingMetrics{Lines: []TextLineSpacing{}, StrokeGaps: [][2]float64{}}
	for i, line := range plan.lines {
		total := textLineGap(line, gap)
		out.Lines = append(out.Lines, TextLineSpacing{plan.widths[i] - total, total})
		anchor := 0.0
		if align == "center" {
			anchor = total / 2
		} else if align == "right" {
			anchor = total
		}
		extra := 0.0
		for index := 0; index < len(line); {
			x := extra - anchor
			if line[index].radical > 0 {
				out.StrokeGaps = append(out.StrokeGaps, [2]float64{x, x}, [2]float64{x, x + float64(line[index].radical)*gap})
				extra += gap
				index++
				continue
			}
			if _, count, _ := f.tableFillLine(line[index:], 0, 0); count > 0 {
				out.StrokeGaps = append(out.StrokeGaps, [2]float64{x, x + float64(count-1)*gap})
				index += count
				extra += float64(count) * gap
				continue
			}
			for _, stroke := range line[index].glyph.Strokes {
				if len(stroke) >= 2 {
					out.StrokeGaps = append(out.StrokeGaps, [2]float64{x, x})
				}
			}
			extra += gap
			if line[index].char == '\t' {
				extra += (tabCells - 1) * gap
			}
			index++
		}
	}
	return out
}
