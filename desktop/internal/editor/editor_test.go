package editor

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func blankProject() project.Project {
	return project.Project{Version: 2, Layout: engine.DefaultLayout(), Writing: engine.DefaultWriting()}
}
func op(tool, params string) Operation { return Operation{tool, json.RawMessage(params)} }
func apply(t *testing.T, p project.Project, ops ...Operation) project.Project {
	t.Helper()
	q, _, err := Apply(p, ops)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestMergeSplitAxesPasteAndSharedBorders(t *testing.T) {
	p := apply(t, blankProject(), op("table.insert", `{"id":"t","rows":2,"cols":2,"width":80}`),
		op("table.paste", `{"id":"t","tsv":"\"实验\n数据\"\t\n甲\t乙"}`),
		op("table.merge", `{"id":"t","selection":{"r0":0,"c0":0,"r1":0,"c1":1}}`),
		op("table.row.insert", `{"id":"t","index":0,"size":12}`),
		op("table.column.insert", `{"id":"t","index":1,"size":15}`))
	merged := p.Tables[0].Cells[cellIndex(&p.Tables[0], 1, 0)]
	if merged.RowSpan != 1 || merged.ColSpan != 3 || merged.Text != "实验\n数据" {
		t.Fatalf("merged topology/text lost: %+v", merged)
	}
	p = apply(t, p, op("table.split", `{"id":"t","selection":{"r0":1,"c0":1,"r1":1,"c1":1}}`),
		op("table.column.delete", `{"id":"t","index":1}`), op("table.row.delete", `{"id":"t","index":0}`),
		op("table.cells", `{"id":"t","selection":{"r0":0,"c0":0,"r1":0,"c1":0},"borders":"none","align":"left"}`))
	tab := p.Tables[0]
	if len(tab.Cells) != 4 || len(tab.RowHeights) != 2 || len(tab.ColumnWidths) != 2 {
		t.Fatal(tab)
	}
	a, b := tab.Cells[0], tab.Cells[1]
	if *a.Borders.Right || *b.Borders.Left || a.Align != "left" {
		t.Fatal("shared edge not hidden")
	}
	p = apply(t, p, op("table.cells", `{"id":"t","borders":"reset"}`), op("table.update", `{"id":"t","page":2,"fontSize":4,"columnWidths":[30,40]}`))
	if p.Tables[0].Page != 1 || p.Tables[0].FontSize != 4 || p.Tables[0].ColumnWidths[1] != 40 {
		t.Fatal(p.Tables[0])
	}
}

func TestCopyMoveResizeTextAndBatchHistoryRetainMinimumHeight(t *testing.T) {
	p := apply(t, blankProject(), op("textbox.insert", `{"id":"caption","text":"表1\n实验数据","x":15,"y":20}`))
	original := Clone(p)
	if original.Tables[0].FontSize != original.Layout.FontSize {
		t.Fatal("new text box must use the document's current font size")
	}
	p = apply(t, p, op("textbox.update", `{"id":"caption","text":"标题","fontSize":4,"align":"center"}`),
		op("object.copy", `{"id":"caption","newId":"copy","page":2,"x":10,"y":15}`),
		op("object.resize", `{"id":"copy","width":90,"height":15}`), op("object.move", `{"id":"copy","page":3}`),
		op("object.delete", `{"id":"caption"}`), op("undo", `{}`), op("redo", `{}`), op("undo", `{}`))
	if len(p.Tables) != 2 || p.Tables[1].Page != 2 || p.Tables[1].ColumnWidths[0] != 90 || p.Tables[1].FontSize != 4 || p.Tables[0].RowHeights[0] != original.Tables[0].RowHeights[0] {
		t.Fatal(p.Tables)
	}
	if !reflect.DeepEqual(original.Tables[0].Cells[0].Text, "表1\n实验数据") {
		t.Fatal("original modified")
	}
	p = apply(t, p, op("writing.set", `{"penUp":8}`), op("undo", `{}`), op("redo", `{}`))
	if *p.Writing.PenUp != 8 || *original.Writing.PenUp != 7 {
		t.Fatal("writing pointer alias changed source/history")
	}
}

func TestTransactionRejectsMalformedOperationsWithoutMutatingSource(t *testing.T) {
	p := apply(t, blankProject(), op("table.insert", `{"id":"t"}`), op("table.cells", `{"id":"t","borders":"all"}`))
	before, _ := json.Marshal(p)
	if p.Tables[0].FontSize != p.Layout.FontSize {
		t.Fatal("new table must use the document's current font size")
	}
	for _, bad := range []Operation{
		op("table.insert", `{"id":"t"}`), op("textbox.insert", `{"rows":2}`), op("table.row.delete", `{"id":"t","index":-1}`),
		op("table.merge", `{"id":"t","selection":{"r0":0,"c0":0,"r1":90,"c1":1}}`),
		op("table.cells", `{"id":"t","align":"diagonal"}`), op("text.set", `{"text":"x","typo":1}`),
		op("text.set", `{}`), op("layout.set", `null`), op("object.move", `{"id":"t","page":0}`), op("undo", `{}`),
	} {
		_, _, err := Apply(p, []Operation{bad})
		if err == nil {
			t.Fatalf("accepted %+v", bad)
		}
		after, _ := json.Marshal(p)
		if string(after) != string(before) {
			t.Fatalf("source mutated on failure: %s", bad.Tool)
		}
	}
	_, _, err := Apply(p, []Operation{op("table.update", `{"id":"t","cells":[{"row":0,"col":0,"rowSpan":1,"colSpan":1,"borders":{"top":false}}]}`)})
	if err == nil {
		t.Fatal("incomplete topology accepted")
	}
	after, _ := json.Marshal(p)
	if string(after) != string(before) {
		t.Fatal("border pointer changed on failed decode")
	}
}

func TestExplicitLinesAndOverflowUseDesktopEngine(t *testing.T) {
	p := apply(t, blankProject(), op("textbox.insert", `{"text":"实验实验实验实验实验实验实验实验","width":3}`))
	f, _ := engine.EmbeddedFont()
	d, err := f.LayoutWithObjects(p.Text, p.Layout, p.Images, p.Tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages[0].Issues) == 0 || d.Pages[0].Tables[0].RowHeights[0] != p.Tables[0].RowHeights[0] {
		t.Fatal("text wrapped automatically")
	}
	_, err = engine.ExportPage(d, 0, p.Writing)
	if err == nil || !strings.Contains(err.Error(), "宽度不足") {
		t.Fatal(err)
	}
}
