package engine

import (
	"fmt"
	"hash/fnv"
	"math"
)

// Each occurrence owns a deterministic stream. Measuring, wrapping and drawing
// in a different order cannot consume randomness or change the next glyph.
type handwritingVariation struct {
	amount float64
	key    uint64
	offset int
}

func (s LayoutSettings) variation(scope string) handwritingVariation {
	h := fnv.New64a()
	h.Write([]byte(scope))
	return handwritingVariation{amount: s.HandwritingVariation / 100, key: h.Sum64() ^ uint64(s.HandwritingSeed)}
}

func imageTextVariation(s LayoutSettings, asset ImageAsset, box OCRBox) handwritingVariation {
	return s.variation(fmt.Sprintf("image:%s:%g:%g:%s", asset.ID, box.X, box.Y, box.Text))
}

func (v handwritingVariation) at(offset int) handwritingVariation {
	v.offset += offset
	return v
}

func (v handwritingVariation) strokes(strokes []Stroke, index int) []Stroke {
	if v.amount <= 0 || len(strokes) == 0 {
		return strokes
	}
	l, r, t, b := tableGlyphBounds(Glyph{Strokes: strokes})
	w, h := r-l, b-t
	// A horizontal/vertical rule has no room for deformation. In particular,
	// preserve tiny punctuation and mathematical rules instead of flattening it.
	if !finite(l) || w < 1e-9 || h < 1e-9 {
		return strokes
	}
	state := v.key + uint64(index+v.offset)*0x9e3779b97f4a7c15
	random := func() float64 {
		state += 0x9e3779b97f4a7c15
		z := state
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		z ^= z >> 31
		return float64(z>>11) / (1 << 53)
	}
	signed := func() float64 { return random()*2 - 1 }
	// The old linear range moved small formula glyphs less than the preview's
	// pen width. Give the lower half of the slider visible shape differences;
	// retain a continuous approach to zero and the same stable random stream.
	a := math.Sqrt(v.amount)
	sx, sy := 1-a*(.02+.22*random()), 1-a*(.02+.20*random())
	short := min(w, h)
	angle := a * .17 * signed() * min(1, short/(max(w, h)*.3))
	shear := a * .20 * signed() * short / h
	bendX, bendY := a*.14*short*signed(), a*.14*short*signed()
	cx, cy := (l+r)/2, (t+b)/2
	sin, cos := math.Sincos(angle)
	out := make([]Stroke, len(strokes))
	for i, stroke := range strokes {
		out[i] = make(Stroke, len(stroke))
		for j, p := range stroke {
			x, y := p.X-cx, p.Y-cy
			nx, ny := 2*x/w, 2*y/h
			// Smooth whole-glyph warp: shared joins stay joined and straight
			// segments remain clean. Never add pointwise jitter or new vertices.
			dx := sx*x + shear*y + bendX*(1-ny*ny)
			dy := sy*y + bendY*(1-nx*nx)
			out[i][j] = Point{cx + cos*dx - sin*dy, cy + sin*dx + cos*dy}
		}
	}
	left, right, top, bottom := tableGlyphBounds(Glyph{Strokes: out})
	fit := min(1, w/(right-left), h/(bottom-top))
	mx, my := (left+right)/2, (top+bottom)/2
	shiftX := (w - (right-left)*fit) / 2 * signed()
	shiftY := (h - (bottom-top)*fit) / 2 * signed()
	maxMove := 0.0
	for i, stroke := range out {
		for j, p := range stroke {
			// Stay in the original ink envelope, so even tightly fitted cells,
			// fractions and edge-of-page text retain their established spacing.
			stroke[j] = Point{max(l, min(r, cx+(p.X-mx)*fit+shiftX)), max(t, min(b, cy+(p.Y-my)*fit+shiftY))}
			maxMove = max(maxMove, math.Hypot(stroke[j].X-strokes[i][j].X, stroke[j].Y-strokes[i][j].Y))
		}
	}
	// Cap the entire deformation together, preserving joins and proportions
	// even when a rare seed combines tilt, shear and bending in one direction.
	limit := .22 * a * math.Hypot(w, h)
	if maxMove > limit {
		for i, stroke := range out {
			for j, p := range stroke {
				q := strokes[i][j]
				stroke[j] = Point{q.X + (p.X-q.X)*limit/maxMove, q.Y + (p.Y-q.Y)*limit/maxMove}
			}
		}
	}
	return out
}
