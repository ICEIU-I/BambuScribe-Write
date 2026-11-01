package engine

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestOCRHyphenStaysInsideItsSourceInterval(t *testing.T) {
	f := testFont(t)
	asset := ImageAsset{Width: 150, Height: 150 * 278. / 1384}
	source := RegionBounds{655. / 1384, .02373407467027918, 17. / 1384, .13138717541591727}
	parent := OCRBox{W: 166.47 / 1384, H: source.Height}
	box := runBox("-", source, parent)
	strokes := handwritingTextStrokes(f, box, asset, DefaultLayout(), 1384, 278)
	if len(strokes) == 0 {
		t.Fatal("hyphen disappeared")
	}
	left, right := math.Inf(1), math.Inf(-1)
	for _, s := range strokes {
		for _, p := range s {
			x := p.X / asset.Width * 1384
			left, right = math.Min(left, x), math.Max(right, x)
		}
	}
	if left < 655-1e-6 || right > 672+1e-6 || right-left < 12 {
		t.Fatalf("17px hyphen occupies %.2f..%.2f", left, right)
	}
}

func TestOCRPolicyDoesNotInventUnverifiedText(t *testing.T) {
	const w, h = 120, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	asset := maskAsset(t, mask, w, h)
	box := OCRBox{Text: "光", X: 3. / w, Y: 6. / h, W: 22. / w, H: 30. / h, Confidence: 1}
	stampStrokes(mask, w, h, handwritingTextStrokes(f, box, asset, DefaultLayout(), w, h), asset, 1)
	for _, tc := range []struct {
		name string
		box  OCRBox
		mode string
	}{
		{"trusted label", box, "handwriting"},
		{"low confidence", OCRBox{Text: "光", X: box.X, Y: box.Y, W: box.W, H: box.H, Confidence: .3}, "keep"},
		{"unsplittable", OCRBox{Text: "光电", X: box.X, Y: box.Y, W: box.W, H: box.H, Confidence: 1}, "keep"},
		{"formula", OCRBox{Text: "UAK/V", X: box.X, Y: box.Y, W: box.W, H: box.H, Confidence: 1}, "keep"},
		{"edge", OCRBox{Text: "10-11", X: -.003, Y: -.01, W: .3, H: .4, Confidence: .3}, "keep"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			regions := resolveImageOCR(f, nil, []OCRBox{tc.box}, mask, mask, w, h, asset)
			if len(regions) != 1 || regions[0].Mode != tc.mode {
				t.Fatalf("unexpected replacement: %+v", regions)
			}
			if err := validateImageRegion(regions[0]); err != nil {
				t.Fatal(err)
			}
		})
	}
	// One valid character beside a misread symbol must not become a mixed label.
	drawSymbol(mask, w, h, 66, h/2, 13)
	box.Text, box.W = "光必", 100./w
	regions := resolveImageOCR(f, nil, []OCRBox{box}, mask, mask, w, h, asset)
	if len(regions) != 1 || regions[0].Mode != "keep" {
		t.Fatalf("partially replaced a rejected label: %+v", regions)
	}
}

func TestOCRRefreshPreservesManualOperationsAndRoundTrips(t *testing.T) {
	f := testFont(t)
	asset := textBoxImage(t)
	old := ImageTextRegion{ID: "old", Text: "-", X: .1, Y: .1, Width: .7, Height: .2, Mode: "handwriting", Source: "ocr", SourceBounds: &RegionBounds{X: .1, Y: .1, Width: .05, Height: .2}}
	erase := ImageTextRegion{ID: "erase", X: .1, Y: .1, Width: .3, Height: .2, Mode: "erase", Source: "manual", ErasePath: []Point{{.1, .1}, {.4, .3}}, EraseRadius: .018}
	manual := ImageTextRegion{ID: "manual", Text: "新文字", X: .2, Y: .4, Width: .3, Height: .2, Mode: "handwriting", Source: "manual"}
	asset.TextRegions = []ImageTextRegion{old, erase, manual}
	if !needsImageOCR(asset.TextRegions) || needsImageOCR([]ImageTextRegion{}) {
		t.Fatal("legacy OCR and explicit no-OCR state confused")
	}
	mask := make([]bool, 200*100)
	for _, boxes := range [][]OCRBox{nil, {{Text: "表1I-UAk关系", X: .1, Y: .1, W: .7, H: .2, Confidence: .3}}} {
		regions := resolveImageOCR(f, asset.TextRegions, boxes, mask, mask, 200, 100, asset)
		if len(regions) != 3 || regions[0].Mode != "keep" || !reflect.DeepEqual(regions[1:], asset.TextRegions[1:]) {
			t.Fatalf("lost or reordered manual edits: %+v", regions)
		}
		encoded, _ := json.Marshal(regions)
		var restored []ImageTextRegion
		if err := json.Unmarshal(encoded, &restored); err != nil || needsImageOCR(restored) {
			t.Fatalf("refreshed regions did not round trip: %v", err)
		}
	}
	// A previously moved OCR object is an intentional edit, not stale geometry.
	old.X = .5
	if needsImageOCR([]ImageTextRegion{old}) {
		t.Fatal("would discard an edited legacy object")
	}
	regions := resolveImageOCR(f, []ImageTextRegion{old}, []OCRBox{{Text: "原文", X: .1, Y: .1, W: .05, H: .2, Confidence: 1}}, mask, mask, 200, 100, asset)
	if len(regions) != 1 || !reflect.DeepEqual(regions[0], old) {
		t.Fatal("duplicated the source of an edited OCR object")
	}
}

func TestKeptFormulaRetainsExponentAndDecimalInk(t *testing.T) {
	const w, h = 120, 60
	mask := make([]bool, w*h)
	for y := 22; y < 45; y++ {
		mask[y*w+15] = true
		mask[y*w+30] = true
	}
	for y := 8; y < 17; y++ {
		mask[y*w+50] = true
		mask[y*w+56] = true
	}
	for x := 39; x <= 45; x++ {
		mask[12*w+x] = true
	}
	mask[43*w+23] = true
	asset := maskAsset(t, mask, w, h)
	asset.TextRegions = []ImageTextRegion{{ID: "formula", Text: "1.0-11", X: .05, Y: .05, Width: .6, Height: .8, Mode: "keep", Source: "ocr", OCRVersion: imageOCRVersion}}
	render, err := imageStrokes(testFont(t), asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]bool, w*h)
	stampStrokes(actual, w, h, render.Strokes, asset, 0)
	for i, ink := range mask {
		if ink && !actual[i] {
			t.Fatalf("lost formula pixel at %d,%d", i%w, i/w)
		}
	}
}

func TestImageEllipsisSurvivesWithoutOCR(t *testing.T) {
	const w, h = 120, 60
	mask := make([]bool, w*h)
	for _, x := range []int{35, 50, 65} {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				mask[(25+dy)*w+x+dx] = true
			}
		}
	}
	mask[40*w+100] = true // an unrelated isolated speck
	asset := maskAsset(t, mask, w, h)
	asset.TextRegions = []ImageTextRegion{}
	render, err := imageStrokes(testFont(t), asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]bool, w*h)
	stampStrokes(actual, w, h, render.Strokes, asset, 0)
	for _, x := range []int{35, 50, 65} {
		if !actual[25*w+x] {
			t.Fatalf("lost ellipsis dot at %d", x)
		}
	}
	if actual[40*w+100] {
		t.Fatal("kept an unrelated isolated speck")
	}
}

func TestAttachedFormulaTableKeepsSourceGeometry(t *testing.T) {
	raw, err := os.ReadFile("testdata/ocr-formula-table.png")
	if err != nil {
		t.Fatal(err)
	}
	const w, h = 1464, 304
	asset := ImageAsset{ID: "table", Width: 150, Height: 150 * float64(h) / w, Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)}
	img, err := decodeImageData(asset.Data)
	if err != nil {
		t.Fatal(err)
	}
	mask := make([]bool, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			mask[y*w+x] = imageSample(img, x, y, w, h) < 230
		}
	}
	// Actual Vision readings from this attachment. Keep recognition deterministic
	// across macOS versions and platforms while exercising the complete renderer.
	boxes := []OCRBox{
		{Text: "表11-UAk关系", X: .4022491339776495, Y: .09539473555091715, W: .16176470772164764, H: .10877193275250885, Confidence: .3},
		{Text: "435.8 mm", X: .05447516600892543, Y: .29107884454074173, W: .0908766574546939, H: .08450897743827412, Confidence: .3},
		{Text: "I/（10-11A）", X: .18322185194472668, Y: .42125057138545907, W: .10846979109967342, H: .1241655161506251, Confidence: .3},
		{Text: "1（10-'A）", X: .18906820673602012, Y: .7360091626367741, W: .09755365965796295, H: .12670430384184195, Confidence: .3},
	}
	dark := append([]bool(nil), mask...)
	rectifiedTableLines(dark, w, h)
	asset.TextRegions = resolveImageOCR(testFont(t), nil, boxes, mask, dark, w, h, asset)
	for _, r := range asset.TextRegions {
		if r.Mode != "keep" {
			t.Fatalf("replaced unverified formula: %+v", r)
		}
	}
	expected := extractImageText(dark, w, h, boxes)
	thinImageMask(expected, w, h)
	render, err := imageStrokes(testFont(t), asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]bool, w*h)
	stampStrokes(actual, w, h, render.Strokes, asset, 0)
	actual = dilateMask(actual, w, h, 1)
	for _, y := range []int{108, 156, 204, 252} {
		for _, x := range []int{892, 909, 925} {
			found := false
			for yy := y - 4; yy <= y+4; yy++ {
				for xx := x - 4; xx <= x+4; xx++ {
					found = found || actual[yy*w+xx]
				}
			}
			if !found {
				t.Fatalf("lost table ellipsis at %d,%d", x, y)
			}
		}
	}
	n, lost := 0, 0
	for i, ink := range expected {
		if ink {
			n++
			if !actual[i] {
				lost++
			}
		}
	}
	if n < 300 || lost != 0 {
		t.Fatalf("lost %d/%d protected source pixels", lost, n)
	}
	// The old long hyphen crossed the otherwise blank gap after the formula.
	for _, s := range render.Strokes {
		for i := 1; i < len(s); i++ {
			if s[i].Y < 60.*asset.Height/h && distance(s[i-1], s[i]) > 40.*asset.Width/w {
				t.Fatal("a long invented segment crosses the title")
			}
		}
	}
}
