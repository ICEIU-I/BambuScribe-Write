package engine

type textSpan struct{ left, right float64 }
type textFragment struct {
	start, end int
	textSpan
	used float64
}
type wrappedTextLine struct {
	parts             []textFragment
	inkTop, inkBottom float64
	hasInk            bool
}

// Body text uses an uninterrupted row. Objects inside its horizontal range
// reserve the entire writing band, including side spaces and gaps. Blank-line
// carets use the same exclusion so typing resumes where the caret is shown.
func availableTextSpans(left, right, y, height float64, objects []RegionBounds) []textSpan {
	if right <= left+1e-8 {
		return nil
	}
	for _, o := range objects {
		if y < o.Y+o.Height+objectLayoutGap-1e-8 && o.Y < y+height+objectLayoutGap-1e-8 &&
			left < o.X+o.Width+objectLayoutGap-1e-8 && o.X < right+objectLayoutGap-1e-8 {
			return nil
		}
	}
	return []textSpan{{left, right}}
}

func (f *Font) fitTextFragment(runes []rune, offset int, available float64, s LayoutSettings) (int, float64) {
	end, used := offset, 0.0
	for end < len(runes) {
		advance := f.advance(runes[end], s)
		if used+advance-s.LetterGap > available+1e-8 {
			break
		}
		used += advance
		end++
	}
	if end < len(runes) && end > offset {
		candidate := end
		for candidate > offset+1 && (noLineStart(f.bodyBase(runes[candidate])) || noLineEnd(f.bodyBase(runes[candidate-1]))) {
			candidate--
		}
		end = candidate
	}
	used = 0
	for _, r := range runes[offset:end] {
		used += f.advance(r, s)
	}
	return end, max(0, used-s.LetterGap)
}

func (f *Font) wrapTextLine(runes []rune, offset int, indent float64, s LayoutSettings, y, pitch float64, objects []RegionBounds) wrappedTextLine {
	left, right := s.MarginLeft+indent, s.PaperWidth-s.MarginRight
	// Selection cells and actual ink both need to clear the objects. If ink
	// extends beyond the cell, widen the band and replan before emitting it.
	top, lower := 0.0, min(s.FontSize, pitch, s.PaperHeight-s.MarginBottom-y)
	for {
		out := wrappedTextLine{}
		at := offset
		for _, span := range availableTextSpans(left, right, y+top, lower-top, objects) {
			end, used := f.fitTextFragment(runes, at, right-left, s)
			if end > at {
				out.parts = append(out.parts, textFragment{at, end, span, used})
				at = end
			}
		}
		out.inkTop, out.inkBottom, out.hasInk = f.lineInkBounds(runes[offset:at], s.FontSize)
		if out.inkTop >= top-1e-8 && out.inkBottom <= lower+1e-8 {
			return out
		}
		top, lower = min(top, out.inkTop), max(lower, out.inkBottom)
	}
}
