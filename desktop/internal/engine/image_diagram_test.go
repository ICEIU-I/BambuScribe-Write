package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func TestRecordedDiagramLabelsAndSingleStrokeExport(t *testing.T) {
	for _, name := range []string{"first", "hall-effect"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("../../docs/diagram-fit-2026-10-09/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			scene, err := diagram.Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
			f := testFont(t)
			if err = f.ValidateDiagram(scene); err != nil {
				t.Fatal(err)
			}
			source := "testdata/hall-switch.png"
			if name == "hall-effect" {
				source = "../../tools/ocrbench/fixtures/hall-effect.png"
			}
			png, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			asset := ImageAsset{ID: "diagram", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), Width: 150, Height: 150 * float64(scene.Height) / float64(scene.Width), Diagram: scene}
			doc, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.Pages[0].Strokes) < len(scene.Strokes) {
				t.Fatal("missing geometry")
			}
			g, err := ExportPage(doc, 0, testWriting())
			if err != nil {
				t.Fatal(err)
			}
			want, err := GenerateGCode(doc.Pages[0], doc.Settings, testWriting())
			if err != nil || string(g) != want {
				t.Fatal("export differs", err)
			}
			if strings.Contains(string(g), "NaN") {
				t.Fatal("invalid coordinates")
			}
			before := len(doc.Pages[0].Strokes)
			asset.TextRegions = []ImageTextRegion{{ID: "erase", Mode: "erase", Source: "manual", X: 0, Y: 0, Width: 1, Height: 1}}
			masked, err := imageStrokes(f, asset, DefaultLayout())
			if err != nil || len(masked.Strokes) != 0 || before == 0 {
				t.Fatal("eraser did not apply to full diagram", err)
			}
		})
	}
}

func TestDiagramMinusCanRender(t *testing.T) {
	f := testFont(t)
	minus := imageRegionTextStrokes(f, OCRBox{Text: "-", W: 1, H: 1}, ImageAsset{Width: 10, Height: 5}, DefaultLayout(), 100, 100)
	if len(minus) == 0 {
		t.Fatal("flat minus glyph vanished")
	}
}
