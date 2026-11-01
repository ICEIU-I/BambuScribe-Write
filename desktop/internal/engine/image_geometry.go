package engine

// Fit long straight runs in source-pixel space. Keep shared endpoints fixed so
// intersecting wires cannot separate, and recurse at actual bends. Short spans
// and curves retain the existing smoother rather than being forced onto axes.
func fitDiagramStroke(s Stroke) Stroke {
	if len(s) < 3 {
		return s
	}
	a, b := s[0], s[len(s)-1]
	worst, split := 0., 0
	for i := 1; i < len(s)-1; i++ {
		if d := pointLineDistance(s[i], a, b); d > worst {
			worst, split = d, i
		}
	}
	chord := distance(a, b)
	if chord >= 24 && worst <= 1.5 && forwardImageStroke(s, a, b) {
		return Stroke{a, b}
	}
	if worst <= 1.5 || split == 0 {
		return smoothImageStroke(s)
	}
	left, right := fitDiagramStroke(s[:split+1]), fitDiagramStroke(s[split:])
	return append(left, right[1:]...)
}

func forwardImageStroke(s Stroke, a, b Point) bool {
	length := distance(a, b)
	if length == 0 {
		return false
	}
	dx, dy := (b.X-a.X)/length, (b.Y-a.Y)/length
	back := 0.
	for i := 1; i < len(s); i++ {
		d := (s[i].X-s[i-1].X)*dx + (s[i].Y-s[i-1].Y)*dy
		if d < 0 {
			back -= d
		}
	}
	return back <= 3
}
