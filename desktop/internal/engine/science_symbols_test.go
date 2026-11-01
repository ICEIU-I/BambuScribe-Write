package engine

import (
	"math"
	"reflect"
	"testing"
)

func TestScienceSymbolsKeepTheirMetricsAndSmoothCurves(t *testing.T) {
	widths := map[rune]float64{'α': 7, 'β': 7, 'γ': 7, 'Δ': 7, 'Ω': 8, 'μ': 4, 'µ': 4, '×': 4, '÷': 4, '±': 4, '−': 5, '≈': 8, '≠': 7, '≤': 7, '≥': 7, '∞': 8, '→': 8, '°': 1.4, '℃': 5.5, '′': 2, '″': 3.5}
	curves := map[rune][]int{'α': {0}, 'γ': {0}, 'Ω': {1}, '≈': {0, 1}, '∞': {0}, '°': {0}, '℃': {1}}
	for _, id := range []string{FontHandwriting, FontDC} {
		f, err := BuiltinFont(id)
		if err != nil {
			t.Fatal(err)
		}
		for r, width := range widths {
			g := f.Glyphs[r]
			if g.Width != width || len(g.Strokes) == 0 {
				t.Fatalf("%s %c: advance or ink lost", id, r)
			}
			for _, s := range g.Strokes {
				for _, p := range s {
					if !finite(p.X) || !finite(p.Y) || p.X < 0 || p.Y < 0 || p.X > f.Units || p.Y > f.Units {
						t.Fatalf("%s %c leaves its cell: %v", id, r, p)
					}
				}
			}
		}
		for r, indices := range curves {
			for _, i := range indices {
				s := f.Glyphs[r].Strokes[i]
				// Round forms must not have the visibly faceted 20–90 degree turns
				// of the old polygons, including the seam of a closed loop.
				if s[0] == s[len(s)-1] {
					s = append(append(Stroke{}, s...), s[1])
				}
				for j := 1; j < len(s)-1; j++ {
					a, b, c := s[j-1], s[j], s[j+1]
					ux, uy, vx, vy := b.X-a.X, b.Y-a.Y, c.X-b.X, c.Y-b.Y
					den := math.Hypot(ux, uy) * math.Hypot(vx, vy)
					if den > 1e-12 && (ux*vx+uy*vy)/den < math.Cos(math.Pi/10) {
						t.Fatalf("%s %c has a sharp turn at point %d", id, r, j)
					}
				}
			}
		}
		for _, pair := range [][2]rune{{'μ', 'µ'}, {'Ω', 'Ω'}, {'Δ', '∆'}} {
			if !reflect.DeepEqual(f.Glyphs[pair[0]], f.Glyphs[pair[1]]) {
				t.Fatal("symbol aliases disagree", pair)
			}
		}
	}
}

func TestScienceOperatorsRetainTheirMeaning(t *testing.T) {
	f := testFont(t)
	g := f.Glyphs['≠']
	horizontal := 0
	for _, s := range g.Strokes {
		if s[0].Y == s[len(s)-1].Y {
			horizontal++
		}
	}
	if horizontal != 2 {
		t.Fatal("not-equal needs both equality bars")
	}
	for _, r := range "≤≥" {
		strokes := f.Glyphs[r].Strokes
		barY := strokes[1][0].Y
		for _, p := range strokes[0] {
			if p.Y >= barY {
				t.Fatal("inequality bar crosses its angle", r)
			}
		}
	}
	for _, r := range "′″" {
		for _, s := range f.Glyphs[r].Strokes {
			if s[0].X-s[len(s)-1].X < .5 {
				t.Fatal("prime looks like a vertical quote", r)
			}
		}
	}
	infinity := f.Glyphs['∞'].Strokes[0]
	if infinity[0] != infinity[len(infinity)-1] {
		t.Fatal("infinity loop is open")
	}
}

func TestSmoothScienceSymbolsPreserveExportedCurves(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID = id
		s.FontSize = 6
		text := "α β γ Δ Ω μ × ÷ ± − ≈ ≠ ≤ ≥ ∞ → ° ℃ ′ ″\n" + `\[R_{Ω}=\frac{α+β}{μ}=\sqrt{γ^{2}+1}\]`
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Missing) > 0 {
			t.Fatal(doc.Missing)
		}
		assertWritingReplay(t, doc)
	}
}
