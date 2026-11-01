package engine

import "math"

// The preview draws round strokes 0.26 mm wide. Two centerlines have visible
// overlap only when their distance is at most the sum of those pen radii.
const bodyInkWidth = 0.26

type inkSegment struct {
	a, b Point
	RegionBounds
}

type textInkGlyph struct {
	RegionBounds
	source      []Stroke
	variation   handwritingVariation
	occurrence  int
	origin      Point
	units, size float64
	segments    []inkSegment
}

// Only glyphs whose envelopes are close need their actual varied strokes.
// Cache those segments for comparisons against later glyphs in nearby rows.
func (g *textInkGlyph) trace() []inkSegment {
	if g.segments != nil {
		return g.segments
	}
	g.segments = []inkSegment{}
	for _, stroke := range g.variation.strokes(g.source, g.occurrence) {
		var previous Point
		for i, p := range stroke {
			point := Point{g.origin.X + p.X/g.units*g.size, g.origin.Y + p.Y/g.units*g.size}
			if i > 0 {
				g.segments = append(g.segments, inkSegment{previous, point, RegionBounds{
					X: min(previous.X, point.X), Y: min(previous.Y, point.Y),
					Width: math.Abs(point.X - previous.X), Height: math.Abs(point.Y - previous.Y),
				}})
			}
			previous = point
		}
	}
	return g.segments
}

func inkBoundsNear(a, b RegionBounds) bool {
	return a.X <= b.X+b.Width+bodyInkWidth && b.X <= a.X+a.Width+bodyInkWidth &&
		a.Y <= b.Y+b.Height+bodyInkWidth && b.Y <= a.Y+a.Height+bodyInkWidth
}

func glyphInkTouches(a, b *textInkGlyph) bool {
	if !inkBoundsNear(a.RegionBounds, b.RegionBounds) {
		return false
	}
	for _, first := range a.trace() {
		for _, second := range b.trace() {
			if inkBoundsNear(first.RegionBounds, second.RegionBounds) && segmentsTouch(first.a, first.b, second.a, second.b) {
				return true
			}
		}
	}
	return false
}

func pointSegmentDistanceSquared(p, a, b Point) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	length := dx*dx + dy*dy
	t := 0.0
	if length > 0 {
		t = max(0, min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/length))
	}
	x, y := p.X-a.X-t*dx, p.Y-a.Y-t*dy
	return x*x + y*y
}

func segmentsTouch(a, b, c, d Point) bool {
	ux, uy, vx, vy := b.X-a.X, b.Y-a.Y, d.X-c.X, d.Y-c.Y
	wx, wy := c.X-a.X, c.Y-a.Y
	denominator := ux*vy - uy*vx
	if math.Abs(denominator) > 1e-12 {
		t := (wx*vy - wy*vx) / denominator
		u := (wx*uy - wy*ux) / denominator
		if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
			return true
		}
	}
	return min(
		pointSegmentDistanceSquared(a, c, d), pointSegmentDistanceSquared(b, c, d),
		pointSegmentDistanceSquared(c, a, b), pointSegmentDistanceSquared(d, a, b),
	) <= bodyInkWidth*bodyInkWidth+1e-12
}
