package diagram

import (
	"math"
	"testing"
)

func TestGeometryKeepsGapsCrossingsAndCircleBoundaries(t *testing.T) {
	s := Scene{Width: 100, Height: 100, Strokes: []Stroke{
		{ID: "left", Points: []Point{{10, 20}, {40, 20}}}, {ID: "right", Points: []Point{{60, 20}, {90, 20}}},
		{ID: "crossing", Points: []Point{{20, 10}, {80, 90}}},
	}, Circles: []Circle{{ID: "contact", CX: 50, CY: 20, R: 5}}}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	p := s.Paths()
	if len(p) != 4 || p[0][1].X != 40 || p[1][0].X != 60 || len(p[2]) != 2 {
		t.Fatal("endpoints changed", p)
	}
	for _, v := range p[3] {
		if math.Abs(math.Hypot(v.X-50, v.Y-20)-5) > 1e-8 {
			t.Fatal("circle deformed")
		}
	}
	s.Strokes[0].Points[0].X = math.NaN()
	if s.Validate() == nil {
		t.Fatal("accepted NaN")
	}
	s.Strokes[0].Points[0].X = -1
	if s.Validate() == nil {
		t.Fatal("accepted outside image")
	}
}

func TestDashedPathDoesNotReconnectGaps(t *testing.T) {
	s := Scene{Strokes: []Stroke{{Points: []Point{{0, 0}, {30, 0}}, Dashed: true}}}
	p := s.Paths()
	if len(p) != 3 || p[0][1].X != 8 || p[1][0].X != 13 || p[1][1].X != 21 || p[2][0].X != 26 {
		t.Fatal(p)
	}
}
