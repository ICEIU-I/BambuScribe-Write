package engine

import (
	"reflect"
	"strings"
	"testing"
)

func sampleTextBox() TableAsset {
	return TableAsset{ID: "caption", Kind: "textbox", Page: 2, X: 12, Y: 18,
		ColumnWidths: []float64{70}, RowHeights: []float64{9},
		Cells: []TableCell{{Row: 0, Col: 0, RowSpan: 1, ColSpan: 1, Text: "表1 I-U关系"}},
		Style: "none", FontSize: 5, Padding: .5, Align: "center", VerticalAlign: "middle"}
}

func TestTextBoxUsesIndependentPageAndIdenticalExportStrokes(t *testing.T) {
	box := sampleTextBox()
	doc, err := testFont(t).LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{box})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 3 || len(doc.Pages[2].Images) != 0 {
		t.Fatal("text box requires an image or loses its page")
	}
	render := doc.Pages[2].Tables[0]
	if render.Kind != "textbox" || len(render.Issues) != 0 || len(render.Strokes) == 0 || !reflect.DeepEqual(render.Strokes, render.Cells[0].Strokes) {
		t.Fatal("caption lost its kind, gained borders, or failed handwriting layout", render.Issues)
	}
	for i, s := range render.Strokes {
		for j, p := range s {
			if doc.Pages[2].Strokes[i][j] != (Point{p.X + box.X, p.Y + box.Y}) {
				t.Fatal("preview and G-code strokes differ")
			}
		}
	}
	if _, err := ExportPage(doc, 2, testWriting()); err != nil {
		t.Fatal(err)
	}
}

func TestTextBoxInvalidContentBlocksOutputAndOverflowMovesToNextPage(t *testing.T) {
	for _, text := range []string{"🙂", strings.Repeat("实验", 100)} {
		box := sampleTextBox()
		box.Cells[0].Text = text
		doc, err := testFont(t).LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{box})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ExportAll(doc, testWriting()); err == nil {
			t.Fatal("invalid text box exported", text)
		}
		if text != "🙂" && doc.Pages[2].Tables[0].Height != box.RowHeights[0] {
			t.Fatal("long text silently wrapped")
		}
	}
	box := sampleTextBox()
	box.Y = 237
	doc, err := testFont(t).LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{box})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 4 || len(doc.Pages[3].Tables) != 1 {
		t.Fatal("caption did not move intact to the next page")
	}
	if _, err := ExportAll(doc, testWriting()); err != nil {
		t.Fatal("resolved caption failed export", err)
	}
}

func TestTextBoxRejectsGridTopologyAndBorders(t *testing.T) {
	box := sampleTextBox()
	if err := box.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []TableAsset{box, box, box, box}
	bad[0].Kind = "unknown"
	bad[1].Style = "grid"
	bad[2].RowHeights = []float64{9, 9}
	bad[3].Cells = append([]TableCell(nil), box.Cells...)
	bad[3].Cells[0].Borders = &TableBorders{}
	for _, candidate := range bad {
		if err := candidate.Validate(); err == nil {
			t.Fatal("text box accepted invalid shape", candidate.Kind)
		}
	}
}
