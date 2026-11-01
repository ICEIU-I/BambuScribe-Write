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

func (f *Font) textBoxSpacing(plan tableText, gap float64, align string) *TextSpacingMetrics {
	out := &TextSpacingMetrics{Lines: []TextLineSpacing{}, StrokeGaps: [][2]float64{}}
	for i, line := range plan.lines {
		total := float64(max(0, len(line)-1)) * gap
		out.Lines = append(out.Lines, TextLineSpacing{plan.widths[i] - total, total})
		anchor := 0.0
		if align == "center" {
			anchor = total / 2
		} else if align == "right" {
			anchor = total
		}
		for index := 0; index < len(line); {
			x := float64(index)*gap - anchor
			if _, count, _ := f.tableFillLine(line[index:], 0, 0); count > 0 {
				out.StrokeGaps = append(out.StrokeGaps, [2]float64{x, x + float64(count-1)*gap})
				index += count
				continue
			}
			for _, stroke := range line[index].glyph.Strokes {
				if len(stroke) >= 2 {
					out.StrokeGaps = append(out.StrokeGaps, [2]float64{x, x})
				}
			}
			index++
		}
	}
	return out
}
