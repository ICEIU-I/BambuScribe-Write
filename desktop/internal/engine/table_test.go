package engine

import (
	"reflect"
	"testing"
)

func sampleTable() TableAsset {
	return TableAsset{ID: "table", Page: 0, X: 12, Y: 20, ColumnWidths: []float64{30, 30}, RowHeights: []float64{10, 10}, Cells: []TableCell{{Row: 0, Col: 0, RowSpan: 1, ColSpan: 1, Text: "实验"}, {Row: 0, Col: 1, RowSpan: 1, ColSpan: 1, Text: "10⁻¹⁹"}, {Row: 1, Col: 0, RowSpan: 1, ColSpan: 1, Text: "数据"}, {Row: 1, Col: 1, RowSpan: 1, ColSpan: 1, Text: "1.602"}}, Style: "grid", FontSize: 4, Padding: 1, Align: "center", VerticalAlign: "middle"}
}
func TestTableSharedEdgesAndMerges(t *testing.T) {
	table := sampleTable()
	lines := tableLines(table, table.RowHeights)
	if len(lines) != 6 {
		t.Fatalf("2x2 grid should contain 3 full horizontal and 3 vertical lines, got %d", len(lines))
	}
	table.Cells = []TableCell{{Row: 0, Col: 0, RowSpan: 1, ColSpan: 2, Text: "标题"}, table.Cells[2], table.Cells[3]}
	if err := table.Validate(); err != nil {
		t.Fatal(err)
	}
	lines = tableLines(table, table.RowHeights)
	for _, s := range lines {
		if s[0].X == 30 && s[1].X == 30 && s[0].Y < 10 {
			t.Fatal("merged header has internal vertical border")
		}
	}
	table.Style = "three-line"
	lines = tableLines(table, table.RowHeights)
	if len(lines) != 3 {
		t.Fatalf("three-line table contains %d lines", len(lines))
	}
	for _, s := range lines {
		if s[0].Y != s[1].Y {
			t.Fatal("three-line table contains vertical rule")
		}
	}
	hidden := false
	table.Cells[0].Borders = &TableBorders{Bottom: &hidden}
	lines = tableLines(table, table.RowHeights)
	if len(lines) != 2 {
		t.Fatal("hidden shared border was redrawn")
	}
}
func TestTableExplicitLinesFitWithoutGrowingCells(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.Cells[0].Text = "实验数据\n需要多行\n保持字号"
	before := append([]float64(nil), table.RowHeights...)
	render := f.renderTable(table, DefaultLayout())
	if len(render.Issues) > 0 {
		t.Fatal(render.Issues)
	}
	if !reflect.DeepEqual(render.RowHeights, table.RowHeights) || render.FontSize >= table.FontSize {
		t.Fatal("multiline content did not fit inside the fixed row")
	}
	if !reflect.DeepEqual(table.RowHeights, before) {
		t.Fatal("preview mutated editable row sizes")
	}
	for _, cell := range render.Cells {
		for _, s := range cell.Strokes {
			for _, p := range s {
				if p.X < cell.X+table.Padding-1e-8 || p.X > cell.X+cell.Width-table.Padding+1e-8 || p.Y < cell.Y+table.Padding-1e-8 || p.Y > cell.Y+cell.Height-table.Padding+1e-8 {
					t.Fatalf("ink %+v outside cell %+v", p, cell)
				}
			}
		}
	}
	table.Cells = []TableCell{{Row: 0, Col: 0, RowSpan: 2, ColSpan: 1, Text: "一\n二\n三\n四\n五"}, table.Cells[1], table.Cells[3]}
	render = f.renderTable(table, DefaultLayout())
	if render.Height != 20 || len(render.Issues) > 0 {
		t.Fatal("spanning content did not fit its region", render)
	}
}
func TestTablesSurviveBodyDeletionAndExportSameStrokes(t *testing.T) {
	table := sampleTable()
	table.Page = 2
	doc, err := testFont(t).LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 3 || len(doc.Pages[2].Tables) != 1 || len(doc.Pages[2].Strokes) == 0 {
		t.Fatal("table-only page lost")
	}
	render := doc.Pages[2].Tables[0]
	for i, s := range render.Strokes {
		for j, p := range s {
			if doc.Pages[2].Strokes[i][j] != (Point{p.X + table.X, p.Y + table.Y}) {
				t.Fatal("preview and output paths differ")
			}
		}
	}
	bytes, err := ExportPage(doc, 2, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	expected, err := GenerateGCode(doc.Pages[2], doc.Settings, testWriting())
	if err != nil || string(bytes) != expected {
		t.Fatal("G-code did not use preview strokes", err)
	}
	edited := table
	edited.Cells = append([]TableCell(nil), table.Cells...)
	edited.Cells[0].Text = "新内容"
	other, err := testFont(t).LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{edited})
	if err != nil {
		t.Fatal(err)
	}
	if other.ID == doc.ID {
		t.Fatal("table edits did not invalidate document ID")
	}
}
func TestTableOverflowMovesToNextPageAndInvalidCellsBlockOutput(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.Y = 235
	doc, err := f.LayoutWithObjects("正文", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 2 || len(doc.Pages[1].Tables) != 1 || len(doc.Pages[1].Issues) != 0 {
		t.Fatal("overflow should move the intact table to the next page")
	}
	if _, err = GenerateGCode(doc.Pages[1], doc.Settings, testWriting()); err != nil {
		t.Fatal("resolved table failed export", err)
	}
	table.Y = 20
	table.Cells[0].Text = "🙂"
	doc, err = f.LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Missing) != 1 || len(doc.Missing) != 1 {
		t.Fatal("table missing glyph was not reported")
	}
	if _, err = GenerateGCode(doc.Pages[0], doc.Settings, testWriting()); err == nil {
		t.Fatal("missing glyph exported")
	}
	table.Cells[0].Text = "汉"
	table.ColumnWidths[0] = 3
	table.FontSize = 8
	doc, err = f.LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Issues) == 0 {
		t.Fatal("oversize character silently crossed a cell border")
	}
}
func TestTableTopologyAndOverlapValidation(t *testing.T) {
	table := sampleTable()
	cases := []TableAsset{table, table, table, table}
	cases[0].Cells = table.Cells[:3]
	cases[1].Cells = append(append([]TableCell(nil), table.Cells...), table.Cells[0])
	cases[2].ColumnWidths = []float64{0, 30}
	cases[3].Cells = append([]TableCell(nil), table.Cells...)
	cases[3].Cells[0].ColSpan = 2
	for i, bad := range cases {
		if err := bad.Validate(); err == nil {
			t.Fatalf("invalid topology %d accepted", i)
		}
	}
	if err := ValidateTables([]TableAsset{table, table}); err == nil {
		t.Fatal("duplicate table ID accepted")
	}
	table.X = 0
	table.Y = 0
	doc, err := testFont(t).LayoutWithObjects("正文", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, doc)
	if len(doc.Pages[0].Warnings) != 0 {
		t.Fatal("resolved overlap still reported a warning")
	}
}

func TestInvisibleTableStillReservesSpaceAndMovesIntact(t *testing.T) {
	table := sampleTable()
	table.Page = 1
	table.Y = 235
	table.Style = "none"
	for i := range table.Cells {
		table.Cells[i].Text = ""
	}
	doc, err := testFont(t).LayoutWithObjects("正文", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 3 || len(doc.Pages[2].Strokes) != 0 || len(doc.Pages[2].Tables) != 1 {
		t.Fatal("invisible table was dropped instead of moved")
	}
	if data, err := ExportAll(doc, testWriting()); err != nil || data == nil {
		t.Fatal("resolved invisible table blocked batch export", err)
	}
}
