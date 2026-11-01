package engine

import "math"

// Curves become short pen movements in the glyph itself, so preview and G-code
// follow the same smooth path. Coordinates are in the font's 11-unit cell.
const symbolCurveStep = .12

// Each three points after the start are two cubic controls and an endpoint.
func symbolCurve(controls Stroke) Stroke {
	out := Stroke{controls[0]}
	for i := 1; i < len(controls); i += 3 {
		a, b, c, d := controls[i-1], controls[i], controls[i+1], controls[i+2]
		length := math.Hypot(b.X-a.X, b.Y-a.Y) + math.Hypot(c.X-b.X, c.Y-b.Y) + math.Hypot(d.X-c.X, d.Y-c.Y)
		n := max(16, int(math.Ceil(length/symbolCurveStep)))
		for j := 1; j <= n; j++ {
			t := float64(j) / float64(n)
			u := 1 - t
			out = append(out, Point{u*u*u*a.X + 3*u*u*t*b.X + 3*u*t*t*c.X + t*t*t*d.X, u*u*u*a.Y + 3*u*u*t*b.Y + 3*u*t*t*c.Y + t*t*t*d.Y})
		}
	}
	return out
}

func symbolEllipse(cx, cy, rx, ry float64) Stroke {
	n := max(32, int(math.Ceil(2*math.Pi*max(rx, ry)/symbolCurveStep)))
	n = (n + 3) / 4 * 4 // Include all four extrema and close with exactly the first point.
	out := make(Stroke, n+1)
	for i := 0; i < n; i++ {
		angle := 2 * math.Pi * float64(i) / float64(n)
		out[i] = Point{cx + rx*math.Cos(angle), cy + ry*math.Sin(angle)}
	}
	out[n] = out[0]
	return out
}
