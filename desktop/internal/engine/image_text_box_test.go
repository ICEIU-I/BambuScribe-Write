package engine

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"reflect"
	"strings"
	"testing"
)

func textBoxImage(t *testing.T) ImageAsset {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for x := 0; x < 200; x++ {
		img.Set(x, 40, color.Black)
	}
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	return ImageAsset{ID: "unit", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data.Bytes()), X: 10, Y: 20, Width: 100, Height: 50, TextRegions: []ImageTextRegion{}}
}
func containsInk(strokes []Stroke, p Point) bool {
	for _, s := range strokes {
		for i := 1; i < len(s); i++ {
			a, b := s[i-1], s[i]
			dx, dy := b.X-a.X, b.Y-a.Y
			dd := dx*dx + dy*dy
			if dd == 0 {
				continue
			}
			u := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / dd
			if u >= 0 && u <= 1 && math.Hypot(a.X+u*dx-p.X, a.Y+u*dy-p.Y) < .3 {
				return true
			}
		}
	}
	return false
}
func TestTextBoxCoverMovesAndDeletionRestoresSource(t *testing.T) {
	f := testFont(t)
	asset := textBoxImage(t)
	before, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	original := before.Pages[0].Images[0].GraphicStrokes
	label := ImageTextRegion{ID: "new", Text: "W/m²", X: .3, Y: .3, Width: .2, Height: .2, Mode: "handwriting", Source: "manual"}
	asset.TextRegions = []ImageTextRegion{label}
	first, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	im := first.Pages[0].Images[0]
	if !reflect.DeepEqual(im.SourceStrokes, original) {
		t.Fatal("lost reusable source ink")
	}
	if containsInk(im.GraphicStrokes, Point{50, 40}) || !containsInk(im.GraphicStrokes, Point{85, 40}) {
		t.Fatal("cover did not clip only its bounds")
	}
	asset.TextRegions[0].X = .65
	moved, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	if !containsInk(moved.Pages[0].Images[0].GraphicStrokes, Point{50, 40}) || containsInk(moved.Pages[0].Images[0].GraphicStrokes, Point{85, 40}) {
		t.Fatal("moving left a hole or failed to move cover")
	}
	// Save/reopen retains object geometry and exact output.
	raw, _ := json.Marshal(asset)
	var restored ImageAsset
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	reopened, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{restored})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(moved.Pages[0].Strokes, reopened.Pages[0].Strokes) {
		t.Fatal("reopening changed output")
	}
	asset.TextRegions = []ImageTextRegion{}
	removed, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(removed.Pages[0].Images[0].GraphicStrokes, original) {
		t.Fatal("deleting textbox did not restore source")
	}
}
func TestTextBoxCoversLowerTextAndExportsOnlyVisibleInk(t *testing.T) {
	f := testFont(t)
	asset := textBoxImage(t)
	asset.TextRegions = []ImageTextRegion{
		{ID: "old", Text: "W/m2", X: .3, Y: .3, Width: .2, Height: .2, Mode: "handwriting", Source: "ocr"},
		{ID: "new", Text: "W/m²", X: .28, Y: .28, Width: .24, Height: .24, Mode: "handwriting", Source: "manual"},
	}
	doc, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	regions := doc.Pages[0].Images[0].TextRegions
	if len(regions[0].Strokes) != 0 || len(regions[0].SourceStrokes) == 0 || len(regions[1].Strokes) == 0 {
		t.Fatal("incorrect text layer order")
	}
	want, err := GenerateGCode(doc.Pages[0], doc.Settings, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	exported, err := ExportPage(doc, 0, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	if string(exported) != want {
		t.Fatal("export differs from visible ink")
	}
}
func TestTextBoxMissingGlyphBlocksOutput(t *testing.T) {
	asset := textBoxImage(t)
	asset.TextRegions = []ImageTextRegion{{ID: "bad", Text: "🙂", X: .3, Y: .3, Width: .2, Height: .2, Source: "manual", Mode: "handwriting"}}
	_, err := testFont(t).LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err == nil || !strings.Contains(err.Error(), "字库不支持") {
		t.Fatalf("missing glyph silently hid source: %v", err)
	}
}
