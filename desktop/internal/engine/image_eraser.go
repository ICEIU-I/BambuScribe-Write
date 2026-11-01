package engine

import (
	"math"
	"sort"
)

type eraseInterval struct{ from, to float64 }

// Erase the final vectors, after line reconstruction and OCR. Changing the
// raster before reconstruction can reconnect erased gaps or move nearby ink.
func eraseImageStrokes(strokes []Stroke, asset ImageAsset, regions []ImageTextRegion) []Stroke {
	for _, region := range regions {
		if region.Source != "manual" || (region.Mode != "erase" && !manualTextBox(region)) {
			continue
		}
		out := make([]Stroke, 0)
		local := func(p Point) Point {
			inverse := asset
			inverse.Rotation = -asset.Rotation
			p = imagePoint(inverse, p.X, p.Y)
			return Point{p.X - asset.X, p.Y - asset.Y}
		}
		brush := make(Stroke, len(region.ErasePath))
		for i, p := range region.ErasePath {
			brush[i] = Point{p.X * asset.Width, p.Y * asset.Height}
		}
		radius := region.EraseRadius
		if radius == 0 {
			radius = .018
		}
		radius *= math.Min(asset.Width, asset.Height)
		for _, stroke := range strokes {
			var path Stroke
			flush := func() {
				if len(path) > 1 {
					out = append(out, path)
				}
				path = nil
			}
			for i := 1; i < len(stroke); i++ {
				originalA, originalB := stroke[i-1], stroke[i]
				a, b := local(originalA), local(originalB)
				var cuts []eraseInterval
				if len(brush) > 0 {
					for j, p := range brush {
						if v, ok := circleInterval(a, b, p, radius); ok {
							cuts = append(cuts, v)
						}
						if j == 0 {
							continue
						}
						q := brush[j-1]
						dx, dy := p.X-q.X, p.Y-q.Y
						length := math.Hypot(dx, dy)
						if length < 1e-10 {
							continue
						}
						transform := func(t Point) Point {
							return Point{((t.X-q.X)*dx + (t.Y-q.Y)*dy) / length, (-(t.X-q.X)*dy + (t.Y-q.Y)*dx) / length}
						}
						if v, ok := rectangleInterval(transform(a), transform(b), 0, -radius, length, radius); ok {
							cuts = append(cuts, v)
						}
					}
				} else if v, ok := rectangleInterval(a, b, region.X*asset.Width, region.Y*asset.Height, (region.X+region.Width)*asset.Width, (region.Y+region.Height)*asset.Height); ok {
					cuts = append(cuts, v)
				}
				sort.Slice(cuts, func(i, j int) bool { return cuts[i].from < cuts[j].from })
				at := func(t float64) Point {
					return Point{originalA.X + t*(originalB.X-originalA.X), originalA.Y + t*(originalB.Y-originalA.Y)}
				}
				add := func(lo, hi float64) {
					if hi-lo < 1e-10 {
						return
					}
					p, q := at(lo), at(hi)
					if len(path) > 0 && distance(path[len(path)-1], p) > 1e-8 {
						flush()
					}
					if len(path) == 0 {
						path = append(path, p)
					}
					path = append(path, q)
				}
				cursor := 0.
				for _, cut := range cuts {
					if cut.from > cursor {
						add(cursor, cut.from)
					}
					if cut.to > cursor {
						flush()
						cursor = cut.to
					}
				}
				if cursor < 1 {
					add(cursor, 1)
				}
			}
			flush()
		}
		strokes = out
	}
	return strokes
}

func rectangleInterval(a, b Point, left, top, right, bottom float64) (eraseInterval, bool) {
	lo, hi := 0., 1.
	for _, s := range [][4]float64{{a.X, b.X - a.X, left, right}, {a.Y, b.Y - a.Y, top, bottom}} {
		if math.Abs(s[1]) < 1e-12 {
			if s[0] < s[2] || s[0] > s[3] {
				return eraseInterval{}, false
			}
			continue
		}
		x, y := (s[2]-s[0])/s[1], (s[3]-s[0])/s[1]
		lo = math.Max(lo, math.Min(x, y))
		hi = math.Min(hi, math.Max(x, y))
		if hi <= lo {
			return eraseInterval{}, false
		}
	}
	return eraseInterval{lo, hi}, true
}

func circleInterval(a, b, center Point, radius float64) (eraseInterval, bool) {
	dx, dy := b.X-a.X, b.Y-a.Y
	x, y := a.X-center.X, a.Y-center.Y
	aa := dx*dx + dy*dy
	bb := 2 * (x*dx + y*dy)
	cc := x*x + y*y - radius*radius
	if aa < 1e-20 {
		return eraseInterval{0, 1}, cc <= 0
	}
	det := bb*bb - 4*aa*cc
	if det <= 0 {
		return eraseInterval{}, false
	}
	root := math.Sqrt(det)
	lo := math.Max(0, (-bb-root)/(2*aa))
	hi := math.Min(1, (-bb+root)/(2*aa))
	return eraseInterval{lo, hi}, hi > lo
}

// Handwritten objects cover only layers below them. OCR replacements keep
// their source-component masking and are never interactive covering objects.
func manualTextBox(region ImageTextRegion) bool {
	return region.Source == "manual" && region.Mode == "handwriting" && region.SourceBounds == nil
}
