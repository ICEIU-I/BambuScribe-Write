package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestDiagramRendersModelSizedLabelsWithoutChangingGeometry(t *testing.T) {
	font := testFont(t)
	scene := &diagram.Scene{
		Width: 200, Height: 100,
		Strokes: []diagram.Stroke{{ID: "wire", Points: []diagram.Point{{X: 10, Y: 20}, {X: 100, Y: 20}}}},
		Labels:  []diagram.Label{{Text: "U_{AK}", X: 25, Y: 45, Width: 30, Height: 10}},
	}
	asset := ImageAsset{Width: 200, Height: 100, Diagram: scene}
	original, err := diagramImageStrokes(font, asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	left, top, right, bottom, ok := inkBounds(original.SourceStrokes[1:])
	if !ok || bottom <= top {
		t.Fatal("source text is missing")
	}
	for _, factor := range []float64{1.5, 1.75, 2} {
		t.Run(fmt.Sprint(factor), func(t *testing.T) {
			scene.Labels[0] = diagram.Label{Text: "U_{AK}", X: 40, Y: 50, Width: 30 * factor, Height: 10 * factor}
			enlarged, err := diagramImageStrokes(font, asset, DefaultLayout())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(original.SourceStrokes[0], enlarged.SourceStrokes[0]) {
				t.Fatal("text enlargement changed the wire")
			}
			l, tp, r, b, ok := inkBounds(enlarged.SourceStrokes[1:])
			if !ok || math.Abs((r-l)/(right-left)-factor) > 1e-9 || math.Abs((b-tp)/(bottom-top)-factor) > 1e-9 {
				t.Fatal("text did not use the model's enlargement factor", factor)
			}
			if l < 40 || tp < 50 || r > 40+30*factor+1e-9 || b > 50+10*factor+1e-9 {
				t.Fatal("text escaped its model-planned placement", l, tp, r, b)
			}
		})
	}
}
