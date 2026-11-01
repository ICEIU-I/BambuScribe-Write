package project

import (
	"archive/zip"
	"bambuscribe/desktop/internal/engine"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"reflect"
	"strings"
	"testing"
)

func pngData(t *testing.T) string {
	t.Helper()
	return encodeImage(t, "png", func(b *bytes.Buffer, img image.Image) error { return png.Encode(b, img) })
}

func jpegData(t *testing.T) string {
	t.Helper()
	return encodeImage(t, "jpeg", func(b *bytes.Buffer, img image.Image) error { return jpeg.Encode(b, img, nil) })
}

func encodeImage(t *testing.T, mime string, encode func(*bytes.Buffer, image.Image) error) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 3; y < 13; y++ {
		for x := 3; x < 13; x++ {
			img.Set(x, y, color.Black)
		}
	}
	var b bytes.Buffer
	if err := encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/" + mime + ";base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

func testProject(t *testing.T) Project {
	t.Helper()
	up, down := 7.0, 3.4
	layout := engine.DefaultLayout()
	layout.TemplateName = "会议记录"
	layout.Align = "center"
	return Project{
		Text:    "第一行\n第二行",
		Layout:  layout,
		Writing: engine.WritingSettings{OffsetX: 1.5, OffsetY: -2, PenUp: &up, PenDown: &down, OriginMode: "manual", Sound: false},
		Images: []engine.ImageAsset{
			{
				ID: "diagram", Page: 1, Data: pngData(t), X: 12, Y: 30, Width: 90, Height: 60, Rotation: 15, Mode: "line",
				TextRegions: []engine.ImageTextRegion{
					{ID: "r1", Text: "压力", X: .1, Y: .2, Width: .3, Height: .1, Mode: "handwriting", Source: "ocr", Confidence: .92},
					{ID: "r2", Text: "", X: .4, Y: .5, Width: .2, Height: .2, Mode: "erase", Source: "manual", ErasePath: []engine.Point{{X: .4, Y: .5}, {X: .6, Y: .7}}, EraseRadius: .04},
				},
			},
			{ID: "photo", Page: 2, Data: jpegData(t), X: 0, Y: 0, Width: 40, Height: 40, Rotation: -90, Mode: "hatch"},
		},
	}
}

func unpack(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string][]byte, len(archive.File))
	for _, file := range archive.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = body
	}
	return entries
}

func TestRoundTripKeepsEveryEditableField(t *testing.T) {
	original := testProject(t)
	data, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Version != Version {
		t.Fatalf("version = %d, want %d", restored.Version, Version)
	}
	if !reflect.DeepEqual(original.Images, restored.Images) {
		t.Fatalf("images changed across the round trip:\n%+v\n%+v", original.Images, restored.Images)
	}
	original.Version, restored.Version = 0, 0
	if !reflect.DeepEqual(original, restored) {
		t.Fatalf("project changed across the round trip:\n%+v\n%+v", original, restored)
	}
}

// The manifest must stay free of image payloads, otherwise the file doubles in
// size and nothing in the archive is readable any more.
func TestArchiveStoresImagesAsOriginalBytes(t *testing.T) {
	original := testProject(t)
	data, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	entries := unpack(t, data)
	manifest, ok := entries[manifestName]
	if !ok {
		t.Fatalf("archive has no %s", manifestName)
	}
	if strings.Contains(string(manifest), "base64") {
		t.Fatal("manifest still carries base64 image data")
	}
	for _, name := range []string{"images/diagram.png", "images/photo.jpeg"} {
		raw, ok := entries[name]
		if !ok {
			t.Fatalf("archive has no %s", name)
		}
		if len(raw) == 0 || string(raw[:1]) == "d" {
			t.Fatalf("%s does not hold raw image bytes", name)
		}
	}
	// A second decode of the same archive must not depend on entry ordering.
	if again, err := Decode(data); err != nil || again.Images[0].ID != "diagram" {
		t.Fatalf("second decode failed: %v", err)
	}
}

func TestDecodeRejectsNewerVersion(t *testing.T) {
	data, err := Encode(testProject(t))
	if err != nil {
		t.Fatal(err)
	}
	entries := unpack(t, data)
	var manifest map[string]any
	if err = json.Unmarshal(entries[manifestName], &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["version"] = Version + 1
	rewritten, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	out := zip.NewWriter(&b)
	if err = writeEntry(out, manifestName, rewritten, zip.Deflate); err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = Decode(b.Bytes()); err == nil || !strings.Contains(err.Error(), "不支持的项目版本") {
		t.Fatalf("expected a version error, got %v", err)
	}
}

func TestDecodeRejectsMissingImageAndBrokenArchive(t *testing.T) {
	data, err := Encode(testProject(t))
	if err != nil {
		t.Fatal(err)
	}
	entries := unpack(t, data)
	var b bytes.Buffer
	out := zip.NewWriter(&b)
	if err = writeEntry(out, manifestName, entries[manifestName], zip.Deflate); err != nil {
		t.Fatal(err)
	}
	if err = writeEntry(out, "images/diagram.png", entries["images/diagram.png"], zip.Store); err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = Decode(b.Bytes()); err == nil || !strings.Contains(err.Error(), "缺少图片 photo") {
		t.Fatalf("expected a missing-image error, got %v", err)
	}
	if _, err = Decode([]byte("not an archive")); err == nil || !strings.Contains(err.Error(), "已损坏") {
		t.Fatalf("expected a corruption error, got %v", err)
	}
}

func TestEncodeRejectsUnusableImages(t *testing.T) {
	duplicate := testProject(t)
	duplicate.Images[1].ID = duplicate.Images[0].ID
	if _, err := Encode(duplicate); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("expected a duplicate-id error, got %v", err)
	}
	escaping := testProject(t)
	escaping.Images[0].ID = "../escape"
	if _, err := Encode(escaping); err == nil || !strings.Contains(err.Error(), "不合法") {
		t.Fatalf("expected an invalid-id error, got %v", err)
	}
	garbage := testProject(t)
	garbage.Images[0].Data = "data:image/png;base64,bm90IGFuIGltYWdl"
	if _, err := Encode(garbage); err == nil || !strings.Contains(err.Error(), "仅支持 PNG 或 JPEG") {
		t.Fatalf("expected an image-format error, got %v", err)
	}
}

func TestEncodeRejectsInvalidLayout(t *testing.T) {
	broken := testProject(t)
	broken.Layout.LineCount = 0
	if _, err := Encode(broken); err == nil {
		t.Fatal("expected a layout error for zero rows")
	}
}

func TestOldHeightCorrectionsAreIgnoredOnOpenAndRemovedOnSave(t *testing.T) {
	original := testProject(t)
	encoded, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	entries := unpack(t, encoded)
	var manifest map[string]any
	if err = json.Unmarshal(entries[manifestName], &manifest); err != nil {
		t.Fatal(err)
	}
	writing := manifest["writing"].(map[string]any)
	writing["bottomLineDrop"] = 2
	writing["bottomLineDrops"] = []float64{2, 3.4, 3.4}
	entries[manifestName], err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	out := zip.NewWriter(&b)
	for name, body := range entries {
		if err = writeEntry(out, name, body, zip.Deflate); err != nil {
			t.Fatal(err)
		}
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	original.Version = Version
	if !reflect.DeepEqual(original, restored) {
		t.Fatal("legacy correction changed other project fields")
	}
	page := engine.Page{Number: 1, LineCount: 16, LineHeight: 15, Strokes: []engine.Stroke{
		{{X: 20, Y: 1}, {X: 20, Y: 210}, {X: 30, Y: 230}},
		{{X: 50, Y: 230}, {X: 40, Y: 1}},
	}}
	code, err := engine.GenerateGCode(page, restored.Layout, restored.Writing)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(code, "G1 Z3.400 ") != 2 || strings.Count(code, "G1 Z7.000 ") != 3 {
		t.Fatal("base pen heights not used for all strokes")
	}
	if strings.Count(code, "G1 Z50.000 ") != 1 || !strings.Contains(code, "G1 Z7.000 F16000\nG1 Z50.000 F16000\nM400\nG1 X") {
		t.Fatal("only the final return should use the higher clearance")
	}
	for _, line := range strings.Split(code, "\n") {
		if strings.HasPrefix(line, "G1 Z") && !strings.HasPrefix(line, "G1 Z3.400 ") && !strings.HasPrefix(line, "G1 Z7.000 ") && !strings.HasPrefix(line, "G1 Z50.000 ") {
			t.Fatalf("unexpected height: %s", line)
		}
	}
	encoded, err = Encode(restored)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(unpack(t, encoded)[manifestName]), "bottomLineDrop") {
		t.Fatal("removed correction persisted")
	}
}
