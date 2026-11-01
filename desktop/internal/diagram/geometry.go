package diagram

import "math"

// Paths samples circles and dashes only. Model endpoints and crossings remain
// unchanged, including separate pen lifts where two lines cross.
func (s *Scene) Paths() [][]Point {
	var out [][]Point
	add := func(points []Point, dashed bool) {
		if !dashed {
			out = append(out, points)
			return
		}
		phase := 0.
		for i := 1; i < len(points); i++ {
			a, b := points[i-1], points[i]
			length := math.Hypot(b.X-a.X, b.Y-a.Y)
			for done := 0.; done < length-1e-8; {
				ink := phase < 8
				boundary := 13.
				if ink {
					boundary = 8
				}
				n := math.Min(boundary-phase, length-done)
				at := func(t float64) Point { return Point{a.X + (b.X-a.X)*t/length, a.Y + (b.Y-a.Y)*t/length} }
				if ink && n > 1e-8 {
					out = append(out, []Point{at(done), at(done + n)})
				}
				done += n
				phase += n
				if phase >= 13-1e-8 {
					phase = 0
				} else if math.Abs(phase-8) < 1e-8 {
					phase = 8
				}
			}
		}
	}
	for _, line := range s.Strokes {
		add(line.Points, line.Dashed)
	}
	for _, c := range s.Circles {
		n := min(2048, max(24, int(math.Ceil(2*math.Pi*c.R/.7))))
		points := make([]Point, n+1)
		for i := range points {
			a := 2 * math.Pi * float64(i) / float64(n)
			points[i] = Point{c.CX + c.R*math.Cos(a), c.CY + c.R*math.Sin(a)}
		}
		points[n] = points[0]
		add(points, c.Dashed)
	}
	return out
}
