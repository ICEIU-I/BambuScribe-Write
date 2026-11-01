package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func fixture() Extraction {
	return Extraction{Tables: []Table{{Title: "表1 U_{AK}关系", Rows: 2, Columns: []float64{2, 1, 1}, Style: "horizontal", Cells: []Cell{{RowSpan: 2, ColSpan: 1, Text: "λ_{i}\nν_{i}"}, {Col: 1, RowSpan: 1, ColSpan: 2, Text: "10¹⁴ Hz"}, {Row: 1, Col: 1, RowSpan: 1, ColSpan: 1, Text: "…"}, {Row: 1, Col: 2, RowSpan: 1, ColSpan: 1, Uncertain: true}}}}}
}
func source() project.Project {
	s := engine.DefaultLayout()
	s.FontSize = 8.21
	return project.Project{Version: 3, Layout: s, Writing: engine.DefaultWriting()}
}
func font(t *testing.T) *engine.Font {
	t.Helper()
	f, e := engine.EmbeddedFont()
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func TestImportPreservesTopologyAndExistingContent(t *testing.T) {
	p := source()
	p.Text = "实验正文"
	p.Tables = []engine.TableAsset{{ID: "original", Page: 0, X: 10, Y: 70, ColumnWidths: []float64{120}, RowHeights: []float64{10}, Cells: []engine.TableCell{{RowSpan: 1, ColSpan: 1}}, Style: "horizontal", FontSize: 8.21, Align: "center", VerticalAlign: "middle"}}
	before, _ := json.Marshal(p)
	plan, err := Build(font(t), p, fixture(), Placement{AfterID: "original"})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(p)
	if string(before) != string(after) {
		t.Fatal("import mutated source")
	}
	a := plan.Objects[len(plan.Objects)-1]
	if a.Kind != "" || a.FontSize != p.Layout.FontSize || a.Y <= 80 || a.Style != "horizontal" || len(a.Cells) != 4 || a.Cells[0].RowSpan != 2 || a.Cells[1].ColSpan != 2 || a.Cells[2].Text != "" || !strings.Contains(a.Cells[0].Text, "_{i}") {
		t.Fatalf("lost topology or formatting: %+v", a)
	}
	if len(plan.Warnings) != 1 {
		t.Fatal(plan.Warnings)
	}
	p.Tables = append(p.Tables, plan.Objects...)
	d, err := font(t).LayoutWithObjects(p.Text, p.Layout, nil, p.Tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages[0].Issues) != 0 || len(d.Pages[0].Warnings) != 0 {
		t.Fatalf("overlap/issues: %+v", d.Pages[0].Warnings)
	}
	b, err := project.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := project.Decode(b)
	if err != nil || !reflect.DeepEqual(restored.Tables, p.Tables) {
		t.Fatal("editable roundtrip failed", err)
	}
	if _, err = engine.ExportPage(d, 0, p.Writing); err != nil {
		t.Fatal(err)
	}
}
func TestImportAvoidsBodyAndContinuesOnNextPage(t *testing.T) {
	p := source()
	p.Text = "正文正文正文\n第二行\n第三行"
	plan, err := Build(font(t), p, fixture(), Placement{Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Objects[0].Y < 40 {
		t.Fatal("did not avoid body", plan.Objects[0].Y)
	}
	plan, err = Build(font(t), p, fixture(), Placement{Page: 1, Y: 235})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range plan.Objects {
		if o.Page != 1 || o.FontSize != 8.21 {
			t.Fatal("wrong page or changed font")
		}
	}
}
func TestImportIsAtomicForInvalidOrTooWideTables(t *testing.T) {
	for _, change := range []func(*Extraction){
		func(e *Extraction) { e.Tables[0].Cells[1].Col = 0 },
		func(e *Extraction) { e.Tables[0].Cells = e.Tables[0].Cells[:3] },
		func(e *Extraction) { e.Tables[0].Columns[0] = math.NaN() },
		func(e *Extraction) { e.Tables[0].Cells[0].Text = strings.Repeat("宽", 2000) },
	} {
		e := fixture()
		change(&e)
		p := source()
		before, _ := json.Marshal(p)
		if _, err := Build(font(t), p, e, Placement{}); err == nil {
			t.Fatal("accepted invalid extraction")
		}
		after, _ := json.Marshal(p)
		if string(before) != string(after) {
			t.Fatal("source changed on failure")
		}
	}
	if _, err := Build(font(t), source(), fixture(), Placement{AfterID: "deleted"}); err == nil {
		t.Fatal("silently ignored stale target")
	}
}
func TestOneLineDoesNotWrapAndMultipleTablesStaySeparate(t *testing.T) {
	e := fixture()
	e.Tables[0].Cells[0].Text = "U_{AK}/V"
	e.Tables = append(e.Tables, e.Tables[0])
	p := source()
	plan, err := Build(font(t), p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(plan.Objects); i++ {
		if plan.Objects[i].Page == plan.Objects[i-1].Page && plan.Objects[i].Y < plan.Objects[i-1].Y+sum(plan.Objects[i-1].RowHeights) {
			t.Fatal("imported tables overlap")
		}
	}
	for _, o := range plan.Objects {
		if o.FontSize != 8.21 {
			t.Fatal("font changed")
		}
	}
}
