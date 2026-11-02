package engine

import "math"

// Spacing uses its own deterministic stream so it never rerolls glyph shapes.
func (v handwritingVariation) spacingRandom(index int, axis uint64) float64 {
	z := v.key ^ axis ^ (uint64(index+v.offset) * 0x9e3779b97f4a7c15)
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11)/(1<<53)*2 - 1
}

// Translate within available whitespace, leaving a buffer next to other ink.
// No cumulative drift: each position is derived from the original layout.
func (v handwritingVariation) spaceOffset(index int, vertical bool, size, before, after float64) float64 {
	if v.amount <= 0 {
		return 0
	}
	axis := uint64(0x4c4554544552)
	if vertical {
		axis = 0x4c494e45
	}
	r := v.spacingRandom(index, axis)
	room := max(0, after)
	if r < 0 {
		room = max(0, before)
	}
	return r * math.Sqrt(v.amount) * min(size*.16, room*.75)
}

func (v handwritingVariation) gap(index int, base float64) float64 {
	if v.amount <= 0 {
		return base
	}
	return base * (1 + .4*math.Sqrt(v.amount)*v.spacingRandom(index, 0x474150))
}

func (f *Font) letterOffset(g Glyph, size, bearing, advance float64, v handwritingVariation, index int) float64 {
	if v.amount <= 0 {
		return 0
	}
	left, right, _, _ := tableGlyphBounds(g)
	if !finite(left) {
		return 0
	}
	return v.spaceOffset(index, false, size, bearing+left/f.Units*size, advance-bearing-right/f.Units*size)
}

// Keep the existing line grid and pagination. Move the whole writing line,
// including its carets, only when its ink still clears preceding text/objects.
func (f *Font) placeVariedTextLine(ink *textInk, runes []rune, line wrappedTextLine, s LayoutSettings, y, pitch float64, objects []RegionBounds, v handwritingVariation) (float64, bool) {
	dy := v.spaceOffset(0, true, s.FontSize, y-s.MarginTop, min(pitch-line.inkBottom, s.PaperHeight-s.MarginBottom-y-max(line.inkBottom, min(s.FontSize, pitch))))
	if line.fittedMath {
		// This equation already fills the row; preserve its pen clearance.
		dy = 0
	}
	candidate := y + dy
	if v.amount > 0 && len(line.parts) > 0 && len(availableTextSpans(s.MarginLeft, s.PaperWidth-s.MarginRight, candidate+min(0, line.inkTop), max(line.inkBottom, min(s.FontSize, pitch))-min(0, line.inkTop), objects)) > 0 {
		trial := textInk{active: append([]textInkGlyph(nil), ink.active...)}
		// Keep the existing conservative placement of varied lines. A rejected
		// spacing trial must not move text simply because validation is finer.
		if trial.addLine(f, runes, line, s, candidate, v) && !trial.envelopeOverlap {
			*ink = trial
			return candidate, true
		}
	}
	return y, ink.addLine(f, runes, line, s, y, v)
}
