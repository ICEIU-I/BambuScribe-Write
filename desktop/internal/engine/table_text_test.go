package engine

import (
	"reflect"
	"testing"
)

func TestTableTrailingWhitespaceDoesNotAddLinesOrMoveInk(t *testing.T) {
	f := testFont(t)
	clean := f.tableText("435.8nm\n光阑2mm", 5, 18)
	for _, suffix := range []string{"       ", "\t\t", "\u3000\u3000\u3000"} {
		padded := f.tableText("435.8nm"+suffix+"\n光阑2mm"+suffix, 5, 18)
		if !reflect.DeepEqual(clean, padded) {
			t.Fatalf("trailing whitespace %q changed line count, width or height", suffix)
		}
	}
}

func TestTableExplicitBlankLinesAndIndentationArePreserved(t *testing.T) {
	f := testFont(t)
	plan := f.tableText("  实验  数据   \n\n结果\n", 4, 100)
	if len(plan.lines) != 4 {
		t.Fatalf("explicit blank lines were lost: %d lines", len(plan.lines))
	}
	if len(plan.lines[0]) != 8 || plan.lines[0][0].char != ' ' || plan.lines[0][4].char != ' ' {
		t.Fatal("leading indentation or internal spacing changed", plan.lines[0])
	}
	if len(plan.lines[1]) != 0 || len(plan.lines[3]) != 0 {
		t.Fatal("explicit blank lines contain phantom spaces")
	}
}

func TestMergedTableCellUsesAvailableHeightForMaximumFont(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.ColumnWidths = []float64{20, 20}
	table.RowHeights = []float64{9.5, 9.5}
	table.FontSize = 5
	table.Cells = []TableCell{
		{Row: 0, Col: 0, RowSpan: 2, ColSpan: 1, Text: "435.8nm       \n光阑2mm"},
		{Row: 0, Col: 1, RowSpan: 1, ColSpan: 1},
		{Row: 1, Col: 1, RowSpan: 1, ColSpan: 1},
	}
	render := f.renderTable(table, DefaultLayout())
	if !reflect.DeepEqual(render.RowHeights, table.RowHeights) {
		t.Fatal("whitespace grew a merged region with enough room", render.RowHeights)
	}
	table.Cells[0].Text = "435.8nm\n光阑2mm"
	clean := f.renderTable(table, DefaultLayout())
	if !reflect.DeepEqual(render.Strokes, clean.Strokes) {
		t.Fatal("trailing whitespace moved the centered handwriting")
	}
	table.Cells[0].Text = "一\n二\n三\n四\n五"
	grown := f.renderTable(table, DefaultLayout())
	if grown.Height != render.Height || grown.FontSize >= render.FontSize {
		t.Fatal("multiline content did not shrink to the merged region")
	}
}

func TestTableOnlyExplicitNewlinesChangeHeight(t *testing.T) {
	f := testFont(t)
	text := "实验数据需要多行显示"
	single := f.tableText(text, 5, 18)
	if len(single.lines) != 1 || single.issue == "" {
		t.Fatal("long text must stay on one line and report insufficient width", single)
	}
	if single.height != f.tableText(text, 5, 100).height {
		t.Fatal("column width changed the height of a single line")
	}
	multiline := f.tableText("实验\n数据\n显示", 5, 18)
	if len(multiline.lines) != 3 || multiline.height <= single.height || multiline.issue != "" {
		t.Fatal("explicit newlines did not create valid separate lines", multiline)
	}
}
