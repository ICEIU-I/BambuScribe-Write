package editor

import (
	"bambuscribe/desktop/internal/engine"
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func lineImage(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.White)
		}
	}
	for x := 5; x < 95; x++ {
		img.Set(x, 25, color.Black)
		img.Set(x, 26, color.Black)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "line.png")
	if err := os.WriteFile(p, b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImageErasureAndCopiesUseRealVectorsAndPreserveSource(t *testing.T) {
	file := lineImage(t)
	raw, _ := json.Marshal(map[string]any{"id": "img", "file": file, "width": 60, "height": 30, "x": 10, "y": 10})
	p := apply(t, blankProject(), Operation{"image.insert", raw})
	f, _ := engine.EmbeddedFont()
	before, err := f.LayoutWithObjects(p.Text, p.Layout, p.Images, nil)
	if err != nil {
		t.Fatal(err)
	}
	p = apply(t, p, op("image.erase.rect", `{"id":"img","regionId":"erase1","x":26,"y":10,"width":8,"height":10}`),
		op("image.erase.path", `{"id":"img","regionId":"erase2","points":[{"x":12,"y":15}],"radius":1}`))
	d, err := f.LayoutWithObjects(p.Text, p.Layout, p.Images, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages[0].Strokes) <= len(before.Pages[0].Strokes) {
		t.Fatal("line was not cut")
	}
	for _, s := range d.Pages[0].Strokes {
		for _, point := range s {
			if point.X > 37 && point.X < 43 && point.Y > 20 && point.Y < 30 {
				t.Fatal("erased region remains in final vectors")
			}
		}
	}
	p = apply(t, p, op("object.copy", `{"id":"img","newId":"copy","page":2,"x":20,"y":20}`),
		op("image.update", `{"id":"copy","rotation":15}`), op("image.erase.delete", `{"id":"copy","regionId":"erase1"}`))
	if len(p.Images[0].TextRegions) != 2 || len(p.Images[1].TextRegions) != 1 || p.Images[1].Rotation != 15 || p.Images[1].Data != p.Images[0].Data {
		t.Fatal("copy lost or altered corrections")
	}
	p = apply(t, p, op("image.text.insert", `{"id":"img","regionId":"label","text":"实验","x":2,"y":2,"width":20,"height":5}`),
		op("image.text.update", `{"id":"img","regionId":"label","text":"数据","x":3}`),
		op("image.text.delete", `{"id":"img","regionId":"label"}`))
	if len(p.Images[0].TextRegions) != 2 {
		t.Fatal("deleting text deleted erasure")
	}
	_, _, err = Apply(p, []Operation{op("image.erase.path", `{"id":"img","points":[{"x":80,"y":1}]}`)})
	if err == nil {
		t.Fatal("out of range brush accepted")
	}
}
