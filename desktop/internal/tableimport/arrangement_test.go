package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"math"
	"testing"
)

func TestManualArrangementPreservesNativePositionsAndRoundtrip(t *testing.T) {
	p := source()
	e := fixture()
	e.Tables = append(e.Tables, e.Tables[0])
	e.Tables[0].Notes = []string{"说明"}
	p.Layout.MarginLeft = 0
	p.Layout.MarginRight = 0
	positions := []Position{{Page: 1, X: 0, Y: 20, Width: 75}, {Page: 1, X: 80, Y: 20, Width: 75}}
	plan, err := Build(font(t), p, e, Placement{Positions: positions})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Groups) != 2 || len(plan.Preview) != 5 {
		t.Fatal(plan)
	}
	for i, g := range plan.Groups {
		if g.Page != positions[i].Page || g.X != positions[i].X || g.Y != positions[i].Y || g.Width > positions[i].Width+1e-7 || len(g.ObjectIDs) < 2 {
			t.Fatal(g)
		}
	}
	p.Tables = plan.Objects
	b, err := project.Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	p, err = project.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := font(t).LayoutWithObjects(p.Text, p.Layout, p.Images, p.Tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 {
		t.Fatal("unexpected overflow")
	}
	for _, o := range doc.Pages[0].Tables {
		for _, want := range plan.Objects {
			if want.ID == o.ID && (math.Abs(o.X-want.X) > 1e-6 || math.Abs(o.Y-want.Y) > 1e-6) {
				t.Fatal("position drift")
			}
		}
	}
	if _, err = engine.ExportPage(doc, 0, p.Writing); err != nil {
		t.Fatal(err)
	}
}
func TestManualArrangementRejectsInvalidBounds(t *testing.T) {
	e := fixture()
	e.Tables = append(e.Tables, e.Tables[0])
	p := source()
	for _, positions := range [][]Position{
		{{Page: 1, X: 0, Y: 0, Width: 75}},
		{{Page: 1, X: -1, Y: 0, Width: 75}, {Page: 2, X: 0, Y: 0, Width: 75}},
		{{Page: 1, X: 0, Y: 230, Width: 75}, {Page: 2, X: 0, Y: 0, Width: 75}},
		{{Page: 1, X: 0, Y: 0, Width: math.NaN()}, {Page: 2, X: 0, Y: 0, Width: 75}},
	} {
		if _, err := Build(font(t), p, e, Placement{Positions: positions}); err == nil {
			t.Fatal("accepted invalid arrangement", positions)
		}
	}
}

func TestManualArrangementAllowsOverlapAndReflowsBody(t *testing.T) {
	p := source()
	p.Text = "正文保持完整。"
	e := fixture()
	e.Tables = append(e.Tables, e.Tables[0])
	positions := []Position{{Page: 1, X: 0, Y: 0, Width: 75}, {Page: 1, X: 0, Y: 0, Width: 75}}
	plan, err := Build(font(t), p, e, Placement{Positions: positions})
	if err != nil {
		t.Fatal(err)
	}
	for i, group := range plan.Groups {
		if group.Page != positions[i].Page || group.X != positions[i].X || group.Y != positions[i].Y || group.Width > positions[i].Width+1e-7 {
			t.Fatal("overlap displaced manual position", group)
		}
	}
	p.Tables = plan.Objects
	doc, err := font(t).LayoutWithObjects(p.Text, p.Layout, p.Images, p.Tables)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Text != p.Text || doc.Pages[0].Cells[0].Y <= plan.Groups[0].Height {
		t.Fatal("body was not moved below overlapping objects")
	}
	if _, err := engine.ExportPage(doc, 0, p.Writing); err != nil {
		t.Fatal(err)
	}
}
