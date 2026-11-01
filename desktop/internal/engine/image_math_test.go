package engine

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestImageFormulaScriptsKeepSizeBaselineAndRotation(t *testing.T) {
	// Matching rectangular glyphs make the relative baselines directly observable.
	glyph := Glyph{Width: 8, Strokes: []Stroke{{{0, 0}, {6, 0}, {6, 10}, {0, 10}}}}
	f := &Font{Units: 10, Glyphs: map[rune]Glyph{'E': glyph, 'H': glyph}}
	asset := ImageAsset{X: 5, Y: 7, Width: 100, Height: 60}
	box := OCRBox{Text: "E_{H}", X: .2, Y: .2, W: .5, H: .6, TextFormat: "math"}
	strokes := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 200, 120)
	if len(strokes) != 2 {
		t.Fatal("formula did not render", strokes)
	}
	height := func(s Stroke) float64 { return s[2].Y - s[0].Y }
	if math.Abs(height(strokes[1])/height(strokes[0])-.65) > 1e-9 || strokes[1][2].Y <= strokes[0][2].Y || strokes[1][0].X <= strokes[0][1].X {
		t.Fatal("subscript was stretched, flattened or overprinted", strokes)
	}
	box.Text = "E^{H}"
	super := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 200, 120)
	if super[1][0].Y >= super[0][0].Y || height(super[1]) >= height(super[0]) {
		t.Fatal("superscript baseline lost")
	}
	box.Text = "E_{H}"
	asset.Rotation = 31
	rotated := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 200, 120)
	for i, s := range strokes {
		for j, p := range s {
			want := imagePoint(asset, p.X, p.Y)
			if distance(want, rotated[i][j]) > 1e-9 {
				t.Fatal("formula did not rotate with image")
			}
		}
	}
	for _, text := range []string{"_H", "E_{}", "E_{H", "E_{H_{H}}", "E_{H}^{H}", `\sqrt{E}`} {
		box.Text = text
		if got := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 200, 120); len(got) != 0 {
			t.Fatalf("unsupported %q erased source", text)
		}
	}
}

func TestHybridDecisionsRefreshAndPersist(t *testing.T) {
	f := testFont(t)
	asset := textBoxImage(t)
	old := ImageTextRegion{ID: "old", Text: "EH", X: .1, Y: .1, Width: .2, Height: .2, Mode: "keep", Source: "ocr", OCRVersion: 2, SourceBounds: &RegionBounds{.1, .1, .2, .2}}
	if !needsImageOCR([]ImageTextRegion{old}) {
		t.Fatal("previous kept OCR never refreshes")
	}
	boxes := []OCRBox{{Text: "E_{H}", X: .1, Y: .1, W: .2, H: .2, Replace: true, Backend: "local-hybrid", TextFormat: "math"}, {Text: "0", X: .6, Y: .1, W: .1, H: .2, Backend: "local-hybrid"}}
	regions := resolveImageOCR(f, []ImageTextRegion{old}, boxes, nil, nil, 200, 100, asset)
	if len(regions) != 2 || regions[0].Mode != "handwriting" || regions[0].TextFormat != "math" || regions[1].Mode != "keep" {
		t.Fatal(regions)
	}
	raw, _ := json.Marshal(regions)
	var restored []ImageTextRegion
	if err := json.Unmarshal(raw, &restored); err != nil || !reflect.DeepEqual(restored, regions) || needsImageOCR(restored) {
		t.Fatal("formula fields lost in saved project", err)
	}
	old.X = .3
	if needsImageOCR([]ImageTextRegion{old}) {
		t.Fatal("moved user object would be refreshed")
	}
}

func TestLocalFailureFallsBackWithoutLosingNativeBoxes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BAMBUSCRIBE_CONFIG_DIR", dir)
	native := []OCRBox{{Text: "原文", X: .1, Y: .1, W: .2, H: .1}}
	got, warning := recognizeLocalOrNative(nil, func() []OCRBox { return native })
	if warning != "" || !reflect.DeepEqual(got, native) {
		t.Fatal("disabled backend changed native OCR")
	}
	os.WriteFile(filepath.Join(dir, "image-ocr.json"), []byte(`{"enabled":true}`), 0600)
	got, warning = recognizeLocalOrNative(nil, func() []OCRBox { return native })
	if warning == "" || !reflect.DeepEqual(got, native) {
		t.Fatal("unavailable local runtime blanked native OCR")
	}
}

func TestRetainedLetterStemIsNotTurnedIntoRule(t *testing.T) {
	const w, h = 140, 90
	mask := make([]bool, w*h)
	for y := 20; y <= 65; y++ {
		mask[y*w+55] = true
	}
	for x := 55; x <= 66; x++ {
		mask[40*w+x] = true
	}
	for y := 40; y <= 65; y++ {
		mask[y*w+66] = true
	}
	for y := 5; y < 85; y++ {
		mask[y*w+85] = true
	}
	asset := maskAsset(t, mask, w, h)
	asset.TextRegions = []ImageTextRegion{{ID: "keep", Text: "h", X: 50. / w, Y: 18. / h, Width: 20. / w, Height: 50. / h, Mode: "keep", Source: "ocr", OCRVersion: imageOCRVersion}}
	render, err := imageStrokes(testFont(t), asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]bool, w*h)
	stampStrokes(actual, w, h, render.Strokes, asset, 0)
	actual = dilateMask(actual, w, h, 1)
	for i, ink := range mask {
		if ink && !actual[i] {
			t.Fatalf("lost protected letter or dimension line at %d,%d", i%w, i/w)
		}
	}
}
