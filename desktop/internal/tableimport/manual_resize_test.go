package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"math"
	"testing"
)

func TestSmartImportCompactsOnceThenAcceptsManualDimensions(t *testing.T) {
	p := source()
	e := Extraction{Tables: []Table{{Rows: 2, Columns: []float64{1, 1}, Style: "grid", Cells: []Cell{
		{Row: 0, Col: 0, RowSpan: 1, ColSpan: 1, Text: "1"},
		{Row: 0, Col: 1, RowSpan: 1, ColSpan: 1, Text: "2"},
		{Row: 1, Col: 0, RowSpan: 1, ColSpan: 1},
		{Row: 1, Col: 1, RowSpan: 1, ColSpan: 1},
	}}}}
	f := font(t)
	plan, err := Build(f, p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	a := plan.Objects[0]
	available := p.Layout.PaperWidth - p.Layout.MarginLeft - p.Layout.MarginRight
	if sum(a.ColumnWidths) >= available*.8 {
		t.Fatal("smart import stretched a short table across the page")
	}
	for i := range a.ColumnWidths {
		a.ColumnWidths[i] += 15
	}
	for i := range a.RowHeights {
		a.RowHeights[i] += 5
	}
	doc, err := f.LayoutWithObjects("", p.Layout, nil, []engine.TableAsset{a})
	if err != nil {
		t.Fatal(err)
	}
	b := doc.Pages[a.Page].Tables[0]
	if math.Abs(b.Width-sum(a.ColumnWidths)) > 1e-8 || math.Abs(b.Height-sum(a.RowHeights)) > 1e-8 {
		t.Fatal("manual dimensions were compacted again", b.Width, b.Height)
	}
}
