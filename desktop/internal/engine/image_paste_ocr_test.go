package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func useFixtureImageOCR(t *testing.T, boxes []OCRBox) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX OCR fixture")
	}
	dir := t.TempDir()
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	encoded, _ := json.Marshal(boxes)
	// The fixture exercises the real configured subprocess path without model downloads.
	script := "#!/bin/sh\ncat > \"$3\" <<'JSON'\n" + string(encoded) + "\nJSON\n"
	write("python", []byte(script))
	for _, name := range []string{"PP-OCRv6_det_medium.onnx", "PP-OCRv6_rec_medium.onnx", "encoder_model.onnx", "decoder_model.onnx"} {
		write(name, []byte("fixture"))
	}
	config, _ := json.Marshal(map[string]any{"enabled": true, "python": filepath.Join(dir, "python"), "formulaPython": filepath.Join(dir, "python"), "modelDir": dir, "formulaModelDir": dir})
	write("image-ocr.json", config)
	t.Setenv("BAMBUSCRIBE_CONFIG_DIR", dir)
}

func TestPastedRasterUsesOCRAndDiagramBypassesIt(t *testing.T) {
	boxes := []OCRBox{{Text: `q\frac{U_{H}}{h}=qvB`, X: 0, Y: 0, W: 1, H: 1, Backend: "local-hybrid", TextFormat: "math", Confidence: .99}}
	useFixtureImageOCR(t, boxes)
	raw, err := os.ReadFile("testdata/hall-fraction.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := ImageAsset{ID: "formula", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw), Width: 80, Height: 80 * 98. / 170}
	f := testFont(t)
	got, err := imageStrokes(f, asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TextRegions) != 1 || got.TextRegions[0].Text != boxes[0].Text || got.TextRegions[0].Mode != "keep" || len(got.Strokes) == 0 {
		t.Fatal("raster OCR did not preserve the unverified fraction", got.TextRegions)
	}
	asset.TextRegions = got.TextRegions
	reopened, err := imageStrokes(f, asset, DefaultLayout())
	if err != nil || !reflect.DeepEqual(got.Strokes, reopened.Strokes) {
		t.Fatal("saved OCR changed strokes", err)
	}
	asset.TextRegions = nil
	asset.Diagram = &diagram.Scene{Width: 170, Height: 98, Strokes: []diagram.Stroke{{ID: "line", Points: []diagram.Point{{X: 10, Y: 10}, {X: 160, Y: 10}}}}}
	vector, err := imageStrokes(f, asset, DefaultLayout())
	if err != nil || len(vector.Strokes) != 1 || len(vector.TextRegions) != 0 {
		t.Fatal("model diagram entered raster OCR", err)
	}
}

func TestRecognizedWholeFractionReplacesRasterAndRefreshesOldRegions(t *testing.T) {
	box := OCRBox{Text: `U_{H}=R_{H}\frac{I_{S}B}{d}`, X: 16. / 148, Y: 18. / 76, W: 109. / 148, H: 47. / 76, Confidence: .947, Backend: "local-hybrid", TextFormat: "math", Replace: true}
	useFixtureImageOCR(t, []OCRBox{box})
	raw, err := os.ReadFile("testdata/hall-voltage-fraction.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := ImageAsset{ID: "fraction", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw), Width: 80, Height: 80 * 76. / 148}
	// A saved automatic region from the previously installed version must refresh.
	asset.TextRegions = []ImageTextRegion{{ID: "old", Text: "IB", X: 94. / 148, Y: 18. / 76, Width: 30. / 148, Height: 20. / 76, Mode: "keep", Source: "ocr", OCRVersion: 3, SourceBounds: &RegionBounds{X: 94. / 148, Y: 18. / 76, Width: 30. / 148, Height: 20. / 76}}}
	f := testFont(t)
	got, err := imageStrokes(f, asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TextRegions) != 1 || got.TextRegions[0].Text != box.Text || got.TextRegions[0].Mode != "handwriting" || len(got.GraphicStrokes) != 0 || !reflect.DeepEqual(got.Strokes, imageFractionStrokes(f, box, asset)) {
		t.Fatal("whole equation did not replace all raster ink", got.TextRegions, len(got.GraphicStrokes))
	}
	asset.TextRegions = got.TextRegions
	if needsImageOCR(asset.TextRegions) {
		t.Fatal("saved formula requests OCR again")
	}
	reopened, err := imageStrokes(f, asset, DefaultLayout())
	if err != nil || !reflect.DeepEqual(got.Strokes, reopened.Strokes) {
		t.Fatal("saved formula strokes changed", err)
	}
}
