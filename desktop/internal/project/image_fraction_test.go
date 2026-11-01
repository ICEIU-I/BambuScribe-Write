package project

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/base64"
	"os"
	"reflect"
	"testing"
)

func TestFractionImageCCFAndGCodeRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../engine/testdata/hall-voltage-fraction.png")
	if err != nil {
		t.Fatal(err)
	}
	p := Project{Version: Version, Layout: engine.DefaultLayout(), Writing: engine.DefaultWriting(), Images: []engine.ImageAsset{{ID: "fraction", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw), Width: 80, Height: 80 * 76. / 148, TextRegions: []engine.ImageTextRegion{{ID: "whole", Text: `U_{H}=R_{H}\frac{I_{S}B}{d}`, X: 16. / 148, Y: 18. / 76, Width: 109. / 148, Height: 47. / 76, Mode: "handwriting", Source: "ocr", OCRVersion: 4, TextFormat: "math", SourceBounds: &engine.RegionBounds{X: 16. / 148, Y: 18. / 76, Width: 109. / 148, Height: 47. / 76}}}}}}
	encoded, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Images, restored.Images) {
		t.Fatal("fraction lost in CCF")
	}
	f, err := engine.EmbeddedFont()
	if err != nil {
		t.Fatal(err)
	}
	var previous string
	for i, p := range []Project{p, restored} {
		doc, err := f.LayoutWithImages(p.Text, p.Layout, p.Images)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Missing) != 0 || len(doc.Pages) != 1 || len(doc.Pages[0].Strokes) == 0 {
			t.Fatal("invalid fraction layout")
		}
		code, err := engine.GenerateGCode(doc.Pages[0], doc.Settings, p.Writing)
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 && code != previous {
			t.Fatal("fraction G-code changed after CCF reopen")
		}
		previous = code
	}
}
