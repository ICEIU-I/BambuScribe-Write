package project

import (
	"bambuscribe/desktop/internal/diagram"
	"bambuscribe/desktop/internal/engine"
	"encoding/base64"
	"os"
	"reflect"
	"testing"
)

func TestDiagramRoundTripPreservesSceneAndRenderedStrokes(t *testing.T) {
	sceneRaw, _ := os.ReadFile("../../docs/diagram-fit-2026-10-09/first.json")
	scene, err := diagram.Decode(sceneRaw)
	if err != nil {
		t.Fatal(err)
	}
	image, _ := os.ReadFile("../engine/testdata/hall-switch.png")
	p := Project{Version: Version, Layout: engine.DefaultLayout(), Writing: engine.DefaultWriting(), Images: []engine.ImageAsset{{ID: "agent", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(image), Width: 150, Height: 58, Diagram: scene}}}
	raw, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Images, p.Images) {
		t.Fatal("diagram not preserved")
	}
	f, _ := engine.EmbeddedFont()
	a, err := f.LayoutWithImages("", p.Layout, p.Images)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.LayoutWithImages("", got.Layout, got.Images)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Pages, b.Pages) {
		t.Fatal("reopen changed vectors")
	}
}
