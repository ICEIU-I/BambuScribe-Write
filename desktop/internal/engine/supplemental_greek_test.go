package engine

import (
	"math"
	"testing"
)

func TestLowercaseLambdaAndNuKeepTheirGreekLetterForms(t *testing.T) {
	f := testFont(t)
	t.Run("lambda forks below its upper-left stem", func(t *testing.T) {
		g := f.Glyphs['λ']
		if g.Width != 7 || len(g.Strokes) != 2 {
			t.Fatal("lambda lost its advance or two connected strokes")
		}
		stem, branch := g.Strokes[0], g.Strokes[1]
		start, join, end := stem[0], branch[0], branch[len(branch)-1]
		if start.Y >= 3 || start.X >= join.X || join.Y <= 3 || join.Y >= 6 || end.X >= join.X || end.Y < 8 {
			t.Fatal("lambda looks like a capital apex instead of a descending stem with a lower-left fork")
		}
		connected := false
		for _, point := range stem {
			if math.Hypot(point.X-join.X, point.Y-join.Y) < .1 {
				connected = true
			}
		}
		if !connected {
			t.Fatal("lambda's fork is detached from its stem")
		}
	})
	t.Run("nu has a rounded bottom instead of a V vertex", func(t *testing.T) {
		g := f.Glyphs['ν']
		if g.Width != 7 || len(g.Strokes) != 1 {
			t.Fatal("nu lost its advance or continuous stroke")
		}
		stroke := g.Strokes[0]
		bottom := 0
		for i, point := range stroke {
			if point.Y > stroke[bottom].Y {
				bottom = i
			}
		}
		if bottom == 0 || bottom == len(stroke)-1 {
			t.Fatal("nu has no rising arc after its bottom")
		}
		a, b, c := stroke[bottom-1], stroke[bottom], stroke[bottom+1]
		incoming, outgoing := Point{b.X - a.X, b.Y - a.Y}, Point{c.X - b.X, c.Y - b.Y}
		turn := (incoming.X*outgoing.X + incoming.Y*outgoing.Y) / (math.Hypot(incoming.X, incoming.Y) * math.Hypot(outgoing.X, outgoing.Y))
		if !finite(turn) || turn < .3 {
			t.Fatal("nu's bottom is a sharp V instead of a rounded turn")
		}
	})
}
