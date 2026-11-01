package engine

import "math"

// Circle outlines touching wires may be split at several shared junctions.
// Recover only closed cycles of at most four arcs with strong circular fit,
// before tangent-based joining can splice an arc into a wire.
func regularizeImageCircles(paths []Stroke) ([]Stroke, []Stroke) {
	used := make([]bool, len(paths))
	adjacent := map[Point][]int{}
	lengths := make([]float64, len(paths))
	for i, s := range paths {
		if len(s) < 2 {
			continue
		}
		lengths[i] = strokeLength(s)
		adjacent[s[0]] = append(adjacent[s[0]], i)
		adjacent[s[len(s)-1]] = append(adjacent[s[len(s)-1]], i)
	}
	circles := []Stroke{}
	anchors := map[Point]Point{}
	accept := func(ids []int, loop Stroke) bool {
		center, radius, ok := fitImageCircle(loop)
		if !ok {
			return false
		}
		circle := Stroke{}
		steps := max(24, int(math.Ceil(2*math.Pi*radius/.8)))
		for i := 0; i <= steps; i++ {
			angle := float64(i) * 2 * math.Pi / float64(steps)
			circle = append(circle, Point{center.X + radius*math.Cos(angle), center.Y + radius*math.Sin(angle)})
		}
		circle[len(circle)-1] = circle[0]
		circles = append(circles, circle)
		for _, id := range ids {
			used[id] = true
			for _, p := range []Point{paths[id][0], paths[id][len(paths[id])-1]} {
				length := distance(center, p)
				if length > 0 {
					anchors[p] = Point{center.X + (p.X-center.X)*radius/length, center.Y + (p.Y-center.Y)*radius/length}
				}
			}
		}
		return true
	}
	for i, s := range paths {
		if used[i] || len(s) < 2 || lengths[i] > 220 {
			continue
		}
		start := s[0]
		var search func([]int, Stroke, float64) bool
		search = func(ids []int, loop Stroke, length float64) bool {
			end := loop[len(loop)-1]
			if end == start {
				return accept(ids, loop)
			}
			if len(ids) >= 4 {
				return false
			}
			for _, j := range adjacent[end] {
				if j <= i || used[j] || length+lengths[j] > 220 {
					continue
				}
				repeated := false
				for _, id := range ids {
					repeated = repeated || id == j
				}
				if repeated {
					continue
				}
				other := paths[j]
				if other[0] != end {
					other = orientedStroke(other, 1)
				}
				next := append(append(Stroke(nil), loop...), other[1:]...)
				if search(append(ids, j), next, length+lengths[j]) {
					return true
				}
			}
			return false
		}
		search([]int{i}, s, lengths[i])
	}
	out := []Stroke{}
	for i, s := range paths {
		if used[i] {
			continue
		}
		p := append(Stroke(nil), s...)
		if a, ok := anchors[p[0]]; ok {
			p[0] = a
		}
		if b, ok := anchors[p[len(p)-1]]; ok {
			p[len(p)-1] = b
		}
		out = append(out, p)
	}
	return out, circles
}

func fitImageCircle(s Stroke) (Point, float64, bool) {
	if len(s) < 12 || distance(s[0], s[len(s)-1]) > .01 {
		return Point{}, 0, false
	}
	points := resampleImageStroke(s, 1)
	points = points[:len(points)-1]
	mean := Point{}
	for _, p := range points {
		mean.X += p.X
		mean.Y += p.Y
	}
	mean.X /= float64(len(points))
	mean.Y /= float64(len(points))
	xx, xy, yy, xr, yr := 0., 0., 0., 0., 0.
	for _, p := range points {
		x, y := p.X-mean.X, p.Y-mean.Y
		q := x*x + y*y
		xx += x * x
		xy += x * y
		yy += y * y
		xr += x * q
		yr += y * q
	}
	det := xx*yy - xy*xy
	if det < 1e-6 {
		return Point{}, 0, false
	}
	c := Point{mean.X + (xr*yy-yr*xy)/(2*det), mean.Y + (yr*xx-xr*xy)/(2*det)}
	radius := 0.
	for _, p := range points {
		radius += distance(c, p)
	}
	radius /= float64(len(points))
	if radius < 3 || radius > 30 {
		return Point{}, 0, false
	}
	squared, worst := 0., 0.
	for _, p := range points {
		d := math.Abs(distance(c, p) - radius)
		squared += d * d
		worst = max(worst, d)
	}
	// Allow bounded blur around the fitted ring. Separate shape checks below
	// preserve squares and ovals; circumference excludes spurs and folded arcs.
	ratio := strokeLength(s) / (2 * math.Pi * radius)
	if worst > min(2, radius*.3) || math.Sqrt(squared/float64(len(points))) > min(.8, radius*.13) || ratio < .9 || ratio > 1.18 {
		return Point{}, 0, false
	}
	bounds, _ := imageInkBounds([]Stroke{s})
	if min(bounds.Width, bounds.Height) < .84*max(bounds.Width, bounds.Height) || circleHasPolygonSide(points, radius) {
		return Point{}, 0, false
	}
	return c, radius, true
}

// Blurred small circles need up to two pixels of radial tolerance. Reject
// squares independently: their long flat sides are not circular raster noise.
func circleHasPolygonSide(points Stroke, radius float64) bool {
	n := len(points)
	for start := 0; start < n; start++ {
		for span := 2; span < n/2; span++ {
			a, b := points[start], points[(start+span)%n]
			if distance(a, b) < radius*1.6 {
				continue
			}
			flat := true
			for step := 1; step < span; step++ {
				if pointLineDistance(points[(start+step)%n], a, b) > .55 {
					flat = false
					break
				}
			}
			if flat {
				return true
			}
		}
	}
	return false
}
