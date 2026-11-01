package engine

import (
	"reflect"
	"testing"
)

func TestScopedErasureKeepsLaterCaptionAndExportsIdenticalInk(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	table := sampleTable()
	table.ID, table.Y = "original", 40
	box := sampleTextBox()
	box.ID, box.Page, box.Y = "later-caption", 0, 40
	ids := []string{table.ID}
	for _, free := range []bool{false, true} {
		e := PageErasure{ID: "old-cut", X: 0, Y: 39, Width: 160, Height: 30, ObjectIDs: &ids}
		if free {
			e.Path = []Point{{0, 50}, {160, 50}}
			e.Radius = 20
		}
		doc, err := f.LayoutWithErasures("", s, nil, []TableAsset{table, box}, []PageErasure{e})
		if err != nil {
			t.Fatal(err)
		}
		caption, err := f.LayoutWithObjects("", s, nil, []TableAsset{box})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(doc.Pages[0].Strokes, caption.Pages[0].Strokes) {
			t.Fatal("old erasure clipped new caption or restored erased original", free)
		}
		if _, err = ExportPage(doc, 0, testWriting()); err != nil {
			t.Fatal(err)
		}
		moved := box
		moved.X += 10
		doc, err = f.LayoutWithErasures("", s, nil, []TableAsset{table, moved}, []PageErasure{e})
		if err != nil || len(doc.Pages[0].Strokes) != len(caption.Pages[0].Strokes) {
			t.Fatal("moving caption through old erasure lost ink", err)
		}
		newIDs := []string{box.ID}
		e2 := e
		e2.ID, e2.ObjectIDs = "new-cut", &newIDs
		doc, err = f.LayoutWithErasures("", s, nil, []TableAsset{table, box}, []PageErasure{e, e2})
		if err != nil || len(doc.Pages[0].Strokes) != 0 {
			t.Fatal("new erasure could not target caption", err)
		}
	}
}

func TestScopedErasureEmptyTargetsOnlyEraseBodyAndValidateIDs(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	box := sampleTextBox()
	box.Page = 0
	ids := []string{}
	e := PageErasure{ID: "body-only", Width: s.PaperWidth, Height: s.PaperHeight, ObjectIDs: &ids}
	doc, err := f.LayoutWithErasures("正文", s, nil, []TableAsset{box}, []PageErasure{e})
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := f.LayoutWithObjects("", s, nil, []TableAsset{box})
	if !reflect.DeepEqual(doc.Pages[0].Strokes, expected.Pages[0].Strokes) {
		t.Fatal("empty targets erased object")
	}
	for _, invalid := range [][]string{{""}, {"same", "same"}} {
		e.ObjectIDs = &invalid
		if ValidateErasures([]PageErasure{e}) == nil {
			t.Fatal("invalid target IDs accepted")
		}
	}
}
