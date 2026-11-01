package engine

import (
	"math"
	"reflect"
	"testing"
)

func TestPaperErasureCutsAllLayersAndKeepsEditableSources(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	image := ImageAsset{ID: "image", Data: testImageData(t), X: 30, Y: 50, Width: 40, Height: 30, Rotation: 25}
	table := sampleTable()
	table.Y = 100
	box := TableAsset{ID: "box", Kind: "textbox", X: 20, Y: 160, ColumnWidths: []float64{90}, RowHeights: []float64{15}, Cells: []TableCell{{RowSpan: 1, ColSpan: 1, Text: "实验"}}, Style: "none", FontSize: 6, Padding: 1, Align: "left", VerticalAlign: "top"}
	other := sampleTable()
	other.ID, other.Page = "other-page", 1
	base, err := f.LayoutWithObjects("正文", s, []ImageAsset{image}, []TableAsset{table, box, other})
	if err != nil {
		t.Fatal(err)
	}
	if len(base.Pages[0].TextStrokes) == 0 || len(base.Pages[0].Images[0].Strokes) == 0 || len(base.Pages[0].Tables[0].Strokes) == 0 || len(base.Pages[0].Tables[1].Strokes) == 0 {
		t.Fatal("fixture has an empty ink layer")
	}
	e := PageErasure{ID: "whole-page", Width: s.PaperWidth, Height: s.PaperHeight}
	doc, err := f.LayoutWithErasures("正文", s, []ImageAsset{image}, []TableAsset{table, box, other}, []PageErasure{e})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Strokes) != 0 {
		t.Fatal("ink survived erasure", doc.Pages[0].Strokes)
	}
	if !reflect.DeepEqual(doc.Pages[0].TextStrokes, base.Pages[0].TextStrokes) || !reflect.DeepEqual(doc.Pages[0].Images, base.Pages[0].Images) || !reflect.DeepEqual(doc.Pages[0].Tables, base.Pages[0].Tables) {
		t.Fatal("erasure mutated source geometry")
	}
	if !reflect.DeepEqual(doc.Pages[1], base.Pages[1]) {
		t.Fatal("erasure changed another page")
	}
	if doc.ID == base.ID {
		t.Fatal("erasure reused an unmasked cached document")
	}
	if _, err = ExportPage(doc, 0, testWriting()); err == nil {
		t.Fatal("erased blank page was exported")
	}
	if _, err = ExportPage(doc, 1, testWriting()); err != nil {
		t.Fatal(err)
	}
	restored, err := f.LayoutWithErasures("正文", s, []ImageAsset{image}, []TableAsset{table, box, other}, nil)
	if err != nil || !reflect.DeepEqual(restored, base) {
		t.Fatal("removing erasure did not restore document", err)
	}
}

func TestPaperFreeEraserKeepsCircularRadiusAndBentPathInterior(t *testing.T) {
	strokes := []Stroke{{{50, 40}, {70, 40}}, {{0, 40}, {100, 40}}, {{50, 80}, {70, 80}}}
	e := PageErasure{ID: "L", Path: []Point{{20, 20}, {20, 60}, {80, 60}}, Radius: 1.8}
	got := erasePageStrokes(strokes, []PageErasure{e})
	if len(got) != 4 || !reflect.DeepEqual(got[0], strokes[0]) || !reflect.DeepEqual(got[3], strokes[2]) {
		t.Fatal("erased outside brush path", got)
	}
	if math.Abs(got[1][1].X-18.2) > 1e-8 || math.Abs(got[2][0].X-21.8) > 1e-8 {
		t.Fatal("radius not measured in paper mm", got)
	}
	got = erasePageStrokes([]Stroke{{{0, 10}, {40, 10}}}, []PageErasure{{X: 16, Y: 8, Width: 8, Height: 4}})
	if len(got) != 2 || got[0][1].X != 16 || got[1][0].X != 24 {
		t.Fatal("rectangle cut failed", got)
	}
}

func TestPaperErasureValidationAndUnusedPages(t *testing.T) {
	for _, e := range []PageErasure{{ID: "x", Page: -1, Width: 1, Height: 1}, {ID: "x", Width: math.NaN(), Height: 1}, {ID: "x", Path: []Point{{math.Inf(1), 1}}, Radius: 1}, {ID: "x", Path: []Point{{1, 1}}, Radius: -1}} {
		if ValidateErasures([]PageErasure{e}) == nil {
			t.Fatal("accepted invalid erasure", e)
		}
	}
	doc, err := testFont(t).LayoutWithErasures("", DefaultLayout(), nil, nil, []PageErasure{{ID: "old", Page: 10, X: 200, Y: 200, Width: 20, Height: 20}})
	if err != nil || len(doc.Pages) != 1 {
		t.Fatal("erasure created unused pages or blocked a resized paper", err)
	}
}
