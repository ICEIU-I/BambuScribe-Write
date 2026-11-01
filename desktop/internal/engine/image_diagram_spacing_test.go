package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestDiagramCaptionsRemovePaddingWithoutRemovingWordOrUnitSpaces(t *testing.T) {
	for _, fontID := range []string{FontHandwriting, FontDC} {
		font, err := BuiltinFont(fontID)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ source, want string }{
			{"图 2 二次成像法光路示意图", "图2二次成像法光路示意图"},
			{"图 3　物 距 像 距 法 光 路 示 意 图", "图3物距像距法光路示意图"},
			{"距离 10 mm", "距离10 mm"},
			{"Light   source 10 mm", "Light source 10 mm"},
		} {
			t.Run(fontID+"/"+tc.source, func(t *testing.T) {
				scene := &diagram.Scene{Width: 700, Height: 400, Labels: []diagram.Label{{Text: tc.source, X: 20, Y: 340, Width: 650, Height: 40}}}
				asset := ImageAsset{Width: 700, Height: 400, Diagram: scene}
				got, err := diagramImageStrokes(font, asset, DefaultLayout())
				if err != nil {
					t.Fatal(err)
				}
				if scene.Labels[0].Text != tc.source {
					t.Fatal("rendering changed the saved recognition")
				}
				scene.Labels[0].Text = tc.want
				want, err := diagramImageStrokes(font, asset, DefaultLayout())
				if err != nil || !reflect.DeepEqual(got.Strokes, want.Strokes) {
					t.Fatal("caption padding changed the rendered text", err)
				}
				if tc.want == "Light source 10 mm" {
					scene.Labels[0].Text = "Lightsource10mm"
					joined, err := diagramImageStrokes(font, asset, DefaultLayout())
					if err != nil || reflect.DeepEqual(got.Strokes, joined.Strokes) {
						t.Fatal("English word or unit separator disappeared", err)
					}
				}
			})
		}
	}
}

func TestDiagramLabelsMatchBodySpacing(t *testing.T) {
	for _, fontID := range []string{FontHandwriting, FontDC} {
		font, err := BuiltinFont(fontID)
		if err != nil {
			t.Fatal(err)
		}
		for _, pitch := range []float64{72.5, 100, 140} {
			for _, gap := range []float64{0, .7} {
				for _, factor := range []float64{1, 2} {
					t.Run(fmt.Sprintf("%s/%g/%g/%g", fontID, pitch, gap, factor), func(t *testing.T) {
						s := DefaultLayout()
						s.PaperWidth, s.PaperHeight, s.FontSize = 1000, 1000, 8*factor
						s.LetterSpacing, s.LetterGap = pitch, gap
						text := "图3物距AB10 mm"
						body, err := font.Layout(text, s)
						if err != nil {
							t.Fatal(err)
						}
						want := body.Pages[0].Strokes
						l, top, r, bottom, _ := inkBounds(want)
						scene := &diagram.Scene{Width: 1000, Height: 1000, Labels: []diagram.Label{{Text: text, X: 10, Y: 10, Width: r - l, Height: bottom - top}}}
						got, err := diagramImageStrokes(font, ImageAsset{Width: 1000, Height: 1000, Diagram: scene}, s)
						if err != nil || len(got.Strokes) != len(want) {
							t.Fatal("diagram did not preserve the body glyphs", err)
						}
						x0, y0, _, _, _ := inkBounds(got.Strokes)
						for i, stroke := range want {
							for j, p := range stroke {
								q := got.Strokes[i][j]
								if math.Hypot((q.X-x0)-(p.X-l), (q.Y-y0)-(p.Y-top)) > 1e-8 {
									t.Fatal("diagram character positions differ from the body, or fixed gaps scaled with the lettering")
								}
							}
						}
					})
				}
			}
		}
	}
}

func TestRecordedCaptionFillsHeightWithoutStretchingTheLine(t *testing.T) {
	font := testFont(t)
	settings := DefaultLayout()
	settings.LetterSpacing, settings.HandwritingVariation = 72.5, 9
	for _, label := range []diagram.Label{
		{Text: "图 2 二次成像法光路示意图", X: 164, Y: 348, Width: 440, Height: 31},
		{Text: "图 3　物距像距法光路示意图", X: 96, Y: 210, Width: 439.5, Height: 28.5},
	} {
		scene := &diagram.Scene{Width: 704, Height: 400, Labels: []diagram.Label{label}}
		got, err := diagramImageStrokes(font, ImageAsset{Width: 704, Height: 400, Diagram: scene}, settings)
		if err != nil {
			t.Fatal(err)
		}
		l, top, r, bottom, ok := inkBounds(got.Strokes)
		if !ok || math.Abs((bottom-top)-label.Height) > 1e-8 || r-l > label.Width+1e-8 {
			t.Fatalf("caption still trades lettering height for blank spacing: %q ink=%gx%g box=%gx%g", label.Text, r-l, bottom-top, label.Width, label.Height)
		}
		if l < label.X || top < label.Y || r > label.X+label.Width || bottom > label.Y+label.Height+1e-8 {
			t.Fatal("caption left its planned box")
		}
	}
}

func TestEnlargedDiagramCaptionUsesActualInkHeight(t *testing.T) {
	font := testFont(t)
	settings := DefaultLayout()
	settings.LetterSpacing, settings.HandwritingVariation = 72.5, 9
	// Both supplied optics captions have 19 px of source ink. A larger final
	// box must enlarge the actual strokes, rather than the character padding.
	for _, text := range []string{"图2二次成像法光路示意图", "图3物距像距法光路示意图"} {
		for _, factor := range []float64{1.5, 1.75, 2} {
			scene := &diagram.Scene{Width: 704, Height: 400, Labels: []diagram.Label{{Text: text, X: 10, Y: 330, Width: 684, Height: 19 * factor}}}
			got, err := diagramImageStrokes(font, ImageAsset{Width: 704, Height: 400, Diagram: scene}, settings)
			if err != nil {
				t.Fatal(err)
			}
			_, top, _, bottom, ok := inkBounds(got.Strokes)
			if !ok || math.Abs((bottom-top)/19-factor) > 1e-8 {
				t.Fatalf("%q requested %gx but actual ink is %gx", text, factor, (bottom-top)/19)
			}
		}
	}
}

func TestDiagramDoesNotShrinkFixedLetterGapToFit(t *testing.T) {
	s := DefaultLayout()
	s.LetterGap = 15
	scene := &diagram.Scene{Width: 200, Height: 100, Labels: []diagram.Label{{Text: "图物", X: 10, Y: 40, Width: 10, Height: 30}}}
	if _, err := diagramImageStrokes(testFont(t), ImageAsset{Width: 200, Height: 100, Diagram: scene}, s); err == nil {
		t.Fatal("an impossible fixed gap disappeared or the label was silently dropped")
	}
}
