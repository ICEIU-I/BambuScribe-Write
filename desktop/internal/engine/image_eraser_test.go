package engine

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestFreeEraserPreservesInteriorOfBentPath(t *testing.T) {
	asset := ImageAsset{Width: 160, Height: 80}
	eraser := ImageTextRegion{ID: "L", Source: "manual", Mode: "erase", X: .2, Y: .2, Width: .6, Height: .6, EraseRadius: .02, ErasePath: []Point{{.2, .2}, {.2, .8}, {.8, .8}}}
	untouched := Stroke{{80, 32}, {112, 32}} // Inside bounding rectangle, outside brush.
	crossed := Stroke{{0, 32}, {160, 32}}
	got := eraseImageStrokes([]Stroke{untouched, crossed}, asset, []ImageTextRegion{eraser})
	if len(got) != 3 || !reflect.DeepEqual(got[0], untouched) {
		t.Fatalf("erased untouched interior: %v", got)
	}
	if math.Abs(got[1][1].X-30.4) > 1e-8 || math.Abs(got[2][0].X-33.6) > 1e-8 {
		t.Fatalf("wrong brush radius: %v", got)
	}
	// Saved data must preserve the same geometry.
	raw, _ := json.Marshal(eraser)
	var restored ImageTextRegion
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, eraseImageStrokes([]Stroke{untouched, crossed}, asset, []ImageTextRegion{restored})) {
		t.Fatal("reopening moved erasure")
	}
}

func TestEraserCircleIsRoundAfterImageResizeAndRotation(t *testing.T) {
	for _, size := range [][2]float64{{160, 80}, {80, 160}} {
		for _, rotation := range []float64{0, 25, 90} {
			asset := ImageAsset{X: 20, Y: 30, Width: size[0], Height: size[1], Rotation: rotation}
			center := Point{asset.X + asset.Width*.5, asset.Y + asset.Height*.5}
			eraser := ImageTextRegion{Source: "manual", Mode: "erase", EraseRadius: .05, ErasePath: []Point{{.5, .5}, {.5, .5}}}
			var strokes []Stroke
			for _, d := range []Point{{1, 0}, {0, 1}} {
				strokes = append(strokes, Stroke{imagePoint(asset, center.X-10*d.X, center.Y-10*d.Y), imagePoint(asset, center.X+10*d.X, center.Y+10*d.Y)})
			}
			got := eraseImageStrokes(strokes, asset, []ImageTextRegion{eraser})
			if len(got) != 4 {
				t.Fatalf("unexpected pieces: %v", got)
			}
			c := imagePoint(asset, center.X, center.Y)
			for i, s := range got {
				p := s[0]
				if i%2 == 0 {
					p = s[len(s)-1]
				}
				if math.Abs(distance(p, c)-4) > 1e-8 {
					t.Fatalf("brush stretched at rotation %v: %v", rotation, p)
				}
			}
		}
	}
}

func TestRectangleErasureClipsCrossingSegments(t *testing.T) {
	got := eraseImageStrokes([]Stroke{{{0, 10}, {40, 10}}}, ImageAsset{Width: 40, Height: 20}, []ImageTextRegion{{Source: "manual", Mode: "erase", X: .4, Y: .4, Width: .2, Height: .2}})
	if len(got) != 2 || math.Abs(got[0][1].X-16) > 1e-8 || math.Abs(got[1][0].X-24) > 1e-8 {
		t.Fatalf("bad rectangle clipping: %v", got)
	}
}

func TestManualErasureCutsOCRAndCanEraseWholeImage(t *testing.T) {
	f := testFont(t)
	asset := ImageAsset{ID: "labels", Data: testImageData(t), X: 20, Y: 20, Width: 50, Height: 50, TextRegions: []ImageTextRegion{
		{ID: "ocr", Text: "P", X: .1, Y: .1, Width: .5, Height: .2, Mode: "handwriting", Source: "ocr"},
		{ID: "erase-all", X: 0, Y: 0, Width: 1, Height: 1, Mode: "erase", Source: "manual"},
	}}
	doc, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Strokes) != 0 || len(doc.Pages[0].Images[0].TextRegions[0].Strokes) != 0 {
		t.Fatal("OCR or diagram survived full erasure")
	}
	// Replacement text added after the eraser must remain printable.
	asset.TextRegions = append(asset.TextRegions, ImageTextRegion{ID: "replacement", Text: "P", X: .1, Y: .1, Width: .5, Height: .2, Mode: "handwriting", Source: "manual"})
	doc, err = f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Strokes) == 0 {
		t.Fatal("erasure hid later replacement text")
	}
	if _, err = ExportPage(doc, 0, testWriting()); err != nil {
		t.Fatal(err)
	}
}
