package engine

// Ink left over from thinning a solid mark, or from a symbol broken by rule
// removal, reaches the pen as specks a fraction of a millimetre long: the head
// taps a dot instead of drawing. Only isolated specks go; a short stroke beside
// other ink is a real detail, such as the dot of a small character or a tick
// against a line.
func dropIsolatedSpecks(strokes []Stroke, minLength, separation float64) []Stroke {
	out := make([]Stroke, 0, len(strokes))
	for i, s := range strokes {
		if strokeLength(s) >= minLength || !isolatedStroke(strokes, i, separation) {
			out = append(out, s)
		}
	}
	return out
}

func isolatedStroke(strokes []Stroke, skip int, separation float64) bool {
	for i, other := range strokes {
		if i == skip {
			continue
		}
		for _, end := range [2]Point{strokes[skip][0], strokes[skip][len(strokes[skip])-1]} {
			if distanceToStroke(end, other) <= separation {
				return false
			}
		}
	}
	return true
}

func distanceToStroke(p Point, s Stroke) float64 {
	if len(s) == 0 {
		return 0
	}
	best := distance(p, s[0])
	for i := 1; i < len(s); i++ {
		best = min(best, pointLineDistance(p, s[i-1], s[i]))
	}
	return best
}
