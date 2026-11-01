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

func TestImportShrinksOnlyOversizedTextBoxes(t *testing.T) {
	for _, field := range []string{"title", "note"} {
		t.Run(field, func(t *testing.T) {
			p := source()
			p.Layout.PaperWidth = 210
			p.Layout.MarginLeft, p.Layout.MarginRight = 25.4, 25.4
			e := fixture()
			long := "I_{M}=1000 mA；霍尔元件置于C型电磁铁气隙中心，X_{c}=________ mm；Y_{c}=________ mm"
			index := 0
			if field == "title" {
				e.Tables[0].Title = long
			} else {
				e.Tables[0].Notes = []string{long, "测量记录"}
				index = 1
			}
			before, _ := json.Marshal(p)
			plan, err := Build(font(t), p, e, Placement{})
			if err != nil {
				t.Fatal(err)
			}
			for i, o := range plan.Objects {
				if i == index {
					if o.FontSize >= p.Layout.FontSize || o.FontSize < 1 || o.Cells[0].Text != long {
						t.Fatalf("long text was not preserved and fitted: %+v", o)
					}
					w, _ := font(t).MeasureTableText(long, o.FontSize)
					if math.Abs(w+2*o.Padding-sum(o.ColumnWidths)) > .001 {
						t.Fatal("text shrank further than necessary")
					}
				} else if o.FontSize != p.Layout.FontSize {
					t.Fatalf("fitting text or table changed font: %+v", o)
				}
			}
			if !strings.Contains(strings.Join(plan.Warnings, " "), "自动缩小") {
				t.Fatal("missing font reduction notice")
			}
			after, _ := json.Marshal(p)
			if string(before) != string(after) {
				t.Fatal("changed original project")
			}
			assertImportRendersAndExports(t, p, plan)
		})
	}
}

func TestImportShrinksWideTableAndPreservesMergedCells(t *testing.T) {
	p := source()
	e := fixture()
	e.Tables[0].Cells[0].Text = strings.Repeat("数据", 15) + "\nU_{AK}/V"
	e.Tables[0].Cells[1].Text = strings.Repeat("频率", 10) + " 10¹⁴ Hz"
	plan, err := Build(font(t), p, e, Placement{Y: 235})
	if err != nil {
		t.Fatal(err)
	}
	a := plan.Objects[len(plan.Objects)-1]
	if a.FontSize >= p.Layout.FontSize || a.FontSize < 1 || plan.Objects[0].FontSize != p.Layout.FontSize {
		t.Fatal("table font was not fitted independently")
	}
	if a.Page != 1 || a.Cells[0].RowSpan != 2 || a.Cells[1].ColSpan != 2 || a.Cells[0].Text != e.Tables[0].Cells[0].Text || a.Cells[1].Text != strings.Repeat("频率", 10)+"10¹⁴ Hz" || a.Cells[2].Text != "" || a.Cells[3].Text != "" {
		t.Fatal("lost placement, merged cells, explicit lines or blank measurements")
	}
	// A slightly larger font must exceed at least one actual cell's width.
	tooWide := false
	for _, c := range a.Cells {
		w, _ := font(t).MeasureTableText(c.Text, a.FontSize+.01)
		tooWide = tooWide || w+2*a.Padding > sum(a.ColumnWidths[c.Col:c.Col+c.ColSpan])
	}
	if !tooWide {
		t.Fatal("table shrank further than necessary")
	}
	assertImportRendersAndExports(t, p, plan)
}

func TestImportRowsUseOnlyMeasuredTextHeight(t *testing.T) {
	p := source()
	e := fixture()
	e.Tables[0].Cells[0].Text = "U_{H}\n10^{-11}"
	plan, err := Build(font(t), p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	a := plan.Objects[len(plan.Objects)-1]
	if math.Abs(sum(a.ColumnWidths)-(p.Layout.PaperWidth-p.Layout.MarginLeft-p.Layout.MarginRight)) > 1e-8 {
		t.Fatal("compact rows changed the import width")
	}
	// Lowering any row above the 3 mm editor minimum must clip at least one
	// cell's measured text, including text spanning several rows.
	for row, height := range a.RowHeights {
		if height <= 3+1e-8 {
			continue
		}
		needed := false
		for _, c := range a.Cells {
			if row < c.Row || row >= c.Row+c.RowSpan {
				continue
			}
			_, h := font(t).MeasureTableText(c.Text, a.FontSize)
			if sum(a.RowHeights[c.Row:c.Row+c.RowSpan])-.01 < h+2*a.Padding {
				needed = true
			}
		}
		if !needed {
			t.Fatalf("row %d retained unnecessary height %v", row, height)
		}
	}
	assertImportRendersAndExports(t, p, plan)
}

func TestImportPreservesMeanVoltageAndWritingBlanks(t *testing.T) {
	p := source()
	e := fixture()
	e.Tables[0].Cells[0].Text = "U\u0305_{H}/mV"
	e.Tables[0].Notes = []string{"I_{M}=1000 mA；霍尔元件置于C型电磁铁气隙中心，X_{c}=________ mm；Y_{c}=________ mm"}
	plan, err := Build(font(t), p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	a := plan.Objects[len(plan.Objects)-1]
	if a.Cells[0].Text != e.Tables[0].Cells[0].Text || plan.Objects[1].Cells[0].Text != e.Tables[0].Notes[0] {
		t.Fatal("import changed the recognized mean voltage or blanks")
	}
	if strings.Contains(strings.Join(plan.Warnings, " "), "缺少") {
		t.Fatal("valid mean bar still reported as a missing glyph", plan.Warnings)
	}
	assertImportRendersAndExports(t, p, plan)
}

func TestImportMinimumFontAndAtomicFailure(t *testing.T) {
	for _, target := range []string{"title", "note", "table"} {
		t.Run(target, func(t *testing.T) {
			p := source()
			e := fixture()
			tooWide := fixture().Tables[0]
			long := strings.Repeat("宽", 400)
			switch target {
			case "title":
				tooWide.Title = long
			case "note":
				tooWide.Notes = []string{long}
			case "table":
				tooWide.Cells[0].Text = long
			}
			e.Tables = append(e.Tables, tooWide)
			before, _ := json.Marshal(p)
			_, err := Build(font(t), p, e, Placement{})
			if err == nil || !strings.Contains(err.Error(), "最小字号 1 mm") {
				t.Fatal("minimum font failure not explained:", err)
			}
			after, _ := json.Marshal(p)
			if string(before) != string(after) {
				t.Fatal("partial import changed original document")
			}
		})
	}
	// A fit exactly at the editor's minimum must still produce a valid object.
	f := font(t)
	text := strings.Repeat("宽", 30)
	w, _ := f.MeasureTableText(text, 1)
	b, err := textBox(f, text, 8.21, w+1, "left")
	if err != nil || b.FontSize != 1 {
		t.Fatalf("minimum boundary failed: %v, %v", b.FontSize, err)
	}
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
}

func assertImportRendersAndExports(t *testing.T, p project.Project, plan Plan) {
	t.Helper()
	p.Tables = append(p.Tables, plan.Objects...)
	data, err := project.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := project.Decode(data)
	if err != nil || !reflect.DeepEqual(restored.Tables, p.Tables) {
		t.Fatal("fitted objects did not survive CCF roundtrip", err)
	}
	doc, err := font(t).LayoutWithObjects(restored.Text, restored.Layout, restored.Images, restored.Tables)
	if err != nil {
		t.Fatal(err)
	}
	for pi, page := range doc.Pages {
		if len(page.Issues) != 0 || len(page.Warnings) != 0 {
			t.Fatalf("page layout failed: %+v, %+v", page.Issues, page.Warnings)
		}
		for _, table := range page.Tables {
			for _, cell := range table.Cells {
				for _, stroke := range cell.Strokes {
					for _, point := range stroke {
						if point.X < cell.X-1e-7 || point.X > cell.X+cell.Width+1e-7 || point.Y < cell.Y-1e-7 || point.Y > cell.Y+cell.Height+1e-7 {
							t.Fatal("fitted ink escaped its cell", point, cell)
						}
					}
				}
			}
		}
		if len(page.Strokes) > 0 {
			if _, err := engine.ExportPage(doc, pi, p.Writing); err != nil {
				t.Fatal(err)
			}
		}
	}
}
