package engine

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestFringeTableContractsWithoutExceedingTheBody(t *testing.T) {
	raw, err := os.ReadFile("testdata/fringe-table.json")
	if err != nil {
		t.Fatal(err)
	}
	var table TableAsset
	if err = json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	original, _ := json.Marshal(table)
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		for _, body := range []float64{6, 8.206666666666667, 11} {
			s := DefaultLayout()
			s.FontID = id
			s.FontSize = body
			compact := f.CompactTable(table, s)
			if tableOffsets(compact.ColumnWidths)[6] >= 176 || tableOffsets(compact.RowHeights)[7] >= tableOffsets(table.RowHeights)[7] {
				t.Fatal("oversized frame did not contract", id, body)
			}
			ratio := compact.ColumnWidths[0] / table.ColumnWidths[0]
			for i, v := range compact.ColumnWidths {
				if math.Abs(v/table.ColumnWidths[i]-ratio) > 1e-8 {
					t.Fatal("column proportions changed")
				}
			}
			for i, v := range compact.RowHeights {
				if math.Abs(v/table.RowHeights[i]-ratio) > 1e-8 {
					t.Fatal("row proportions changed")
				}
			}
			again := f.CompactTable(compact, s)
			if !reflect.DeepEqual(compact, again) {
				t.Fatal("table keeps contracting on redraw")
			}
			doc, err := f.LayoutWithObjects("", s, nil, []TableAsset{compact})
			if err != nil {
				t.Fatal(err)
			}
			render := doc.Pages[0].Tables[0]
			for _, c := range render.Cells {
				if c.Text != "" && (c.FontSize > body+1e-7 || c.FontSize < body*.7-1e-7) {
					t.Fatal("font outside 70–100 percent", id, body, c.Text, c.FontSize)
				}
			}
			if _, err = ExportPage(doc, 0, testWriting()); err != nil {
				t.Fatal(err)
			}
		}
	}
	after, _ := json.Marshal(table)
	if string(after) != string(original) {
		t.Fatal("render mutated editable source")
	}
}

func TestOversizedSingleCellCompactsOnceAndKeepsEditorMinimums(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	a := TableAsset{ID: "large", ColumnWidths: []float64{900}, RowHeights: []float64{900}, Cells: []TableCell{{RowSpan: 1, ColSpan: 1, Text: "1"}}, Style: "grid", FontSize: 40, Padding: 1, Align: "center", VerticalAlign: "middle"}
	b := f.CompactTable(a, s)
	if !reflect.DeepEqual(b, f.CompactTable(b, s)) {
		t.Fatal("large table contracts on every render")
	}
	if b.ColumnWidths[0] < 3 || b.RowHeights[0] < 3 {
		t.Fatal("compaction violated editor minimums")
	}
	render := f.renderTable(b, s)
	if render.Cells[0].FontSize > 6+1e-7 || render.Width >= 900 {
		t.Fatal("large frame or font retained")
	}
}
