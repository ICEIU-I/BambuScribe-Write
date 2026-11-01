package engine

import (
	"math"
	"reflect"
	"testing"
)

func TestManualTableSizeSurvivesLayoutAndExport(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, err := BuiltinFont(id)
		if err != nil {
			t.Fatal(err)
		}
		s := DefaultLayout()
		s.FontID = id
		s.FontSize = 6
		a := sampleTable()
		a.ColumnWidths = []float64{55, 55}
		a.RowHeights = []float64{20, 20}
		for i := range a.Cells {
			a.Cells[i].Text = "1"
		}
		doc, err := f.LayoutWithObjects("", s, nil, []TableAsset{a})
		if err != nil {
			t.Fatal(err)
		}
		b := doc.Pages[0].Tables[0]
		if math.Abs(b.Width-110) > 1e-8 || math.Abs(b.Height-40) > 1e-8 ||
			!reflect.DeepEqual(a.ColumnWidths, b.ColumnWidths) || !reflect.DeepEqual(a.RowHeights, b.RowHeights) {
			t.Fatalf("manual grid changed: %s, %.3f x %.3f", id, b.Width, b.Height)
		}
		if _, err = ExportPage(doc, 0, testWriting()); err != nil {
			t.Fatal(err)
		}
		if len(b.Issues) != 0 || len(doc.Missing) != 0 {
			t.Fatal("manual table failed layout", b.Issues, doc.Missing)
		}
	}
}
