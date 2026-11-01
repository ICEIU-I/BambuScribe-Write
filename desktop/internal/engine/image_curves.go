package engine

import "math"

func strokeLength(s Stroke) float64 {
	length := 0.
	for i := 1; i < len(s); i++ {
		length += distance(s[i-1], s[i])
	}
	return length
}
func distance(a, b Point) float64          { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func mixPoint(a, b Point, t float64) Point { return Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t} }
func pointLineDistance(p, a, b Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	if dx == 0 && dy == 0 {
		return distance(p, a)
	}
	t := max(0, min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/(dx*dx+dy*dy)))
	return distance(p, Point{a.X + t*dx, a.Y + t*dy})
}
func simplifyImageStroke(s Stroke, tolerance float64) Stroke {
	if len(s) < 3 {
		return s
	}
	keep := make([]bool, len(s))
	keep[0], keep[len(s)-1] = true, true
	stack := [][2]int{{0, len(s) - 1}}
	for len(stack) > 0 {
		r := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		split := -1
		worst := tolerance
		for i := r[0] + 1; i < r[1]; i++ {
			if d := pointLineDistance(s[i], s[r[0]], s[r[1]]); d > worst {
				worst, split = d, i
			}
		}
		if split >= 0 {
			keep[split] = true
			stack = append(stack, [2]int{r[0], split}, [2]int{split, r[1]})
		}
	}
	out := Stroke{}
	for i, p := range s {
		if keep[i] {
			out = append(out, p)
		}
	}
	return out
}

// Operate in source-pixel space so resizing a placed image doesn't change its
// geometry. First suppress raster stair steps, then fit between sparse knots.
func smoothImageStroke(s Stroke) Stroke {
	if len(s) < 3 {
		return s
	}
	filtered := append(Stroke(nil), s...)
	for i := 2; i < len(s)-2; i++ {
		a, b := s[i-2], s[i+2]
		u, v := Point{s[i].X - a.X, s[i].Y - a.Y}, Point{b.X - s[i].X, b.Y - s[i].Y}
		if u.X*v.X+u.Y*v.Y < .5*distance(a, s[i])*distance(s[i], b) {
			continue
		} // retain corners
		filtered[i] = Point{(s[i-2].X + 2*s[i-1].X + 3*s[i].X + 2*s[i+1].X + s[i+2].X) / 9, (s[i-2].Y + 2*s[i-1].Y + 3*s[i].Y + 2*s[i+1].Y + s[i+2].Y) / 9}
	}
	knots := simplifyImageStroke(filtered, .65)
	if len(knots) < 3 {
		return knots
	}
	// Two corner-cutting passes stay in the source hull (no cubic overshoot at
	// endpoints or grid corners). Endpoints, junctions and sharp corners survive.
	for pass := 0; pass < 2; pass++ {
		out := Stroke{knots[0]}
		for i := 1; i < len(knots)-1; i++ {
			a, b, c := knots[i-1], knots[i], knots[i+1]
			den := distance(a, b) * distance(b, c)
			if den == 0 || ((b.X-a.X)*(c.X-b.X)+(b.Y-a.Y)*(c.Y-b.Y))/den < .65 {
				out = append(out, b)
				continue
			}
			out = append(out, mixPoint(b, a, .22), mixPoint(b, c, .22))
		}
		out = append(out, knots[len(knots)-1])
		knots = out
	}
	return simplifyImageStroke(knots, .06)
}

// Equal arc lengths, including through bends; gaps are actual pen-up moves.
func dashImageStroke(path Stroke, ink, gap float64) []Stroke {
	if len(path) < 2 || ink <= 0 || gap <= 0 {
		return nil
	}
	out := []Stroke{}
	current := Stroke{path[0]}
	remaining := ink
	drawing := true
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		length := distance(a, b)
		for length > 1e-8 {
			step := math.Min(length, remaining)
			end := mixPoint(a, b, step/length)
			if drawing {
				current = append(current, end)
			}
			remaining -= step
			length -= step
			a = end
			if remaining < 1e-8 {
				if drawing {
					if strokeLength(current) > 1e-6 {
						out = append(out, current)
					}
					current = nil
					remaining = gap
				} else {
					current = Stroke{a}
					remaining = ink
				}
				drawing = !drawing
			}
		}
	}
	if drawing && strokeLength(current) > .01 {
		out = append(out, current)
	}
	return out
}
