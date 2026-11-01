package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestTableSubscriptKeepsCapitalShapeAndLowersInk(t *testing.T) {
	f := testFont(t)
	plan := f.tableText("U_{AK}/V", 5, 100)
	var chars strings.Builder
	for _, letter := range plan.lines[0] {
		chars.WriteRune(letter.char)
	}
	if len(plan.lines) != 1 || chars.String() != "UAK/V" || plan.issue != "" {
		t.Fatal("subscript markers became visible or created a new line", plan)
	}
	plain := f.tableText("UAK/V", 5, 100)
	if plan.widths[0] >= plain.widths[0] {
		t.Fatal("subscript letters still occupy full-size width")
	}
	for _, r := range "AK" {
		normal := f.tableText(string(r), 5, 100)
		sub := f.tableText("_{"+string(r)+"}", 5, 100)
		body, missing := f.placeTableText(normal, 0, 0, 0, 100, 10, "left", "top")
		script, missingSub := f.placeTableText(sub, 0, 0, 0, 100, 10, "left", "top")
		if len(missing) != 0 || len(missingSub) != 0 || len(body) != len(script) {
			t.Fatal("capital subscript lost original strokes", r)
		}
		for i, stroke := range body {
			for j, p := range stroke {
				q := script[i][j]
				// Restore the common font origin: each standalone cell now
				// aligns its actual ink to the top padding.
				if math.Abs(q.X-.65*p.X) > 1e-8 || math.Abs(q.Y+sub.inkTop-(.65*(p.Y+normal.inkTop)+2)) > 1e-8 {
					t.Fatalf("%c subscript must scale both axes equally and move down: body=%+v sub=%+v", r, p, q)
				}
			}
		}
	}
	for _, i := range []int{0, 3, 4} {
		if !reflect.DeepEqual(plan.lines[0][i], plain.lines[0][i]) {
			t.Fatal("subscript changed surrounding normal glyphs")
		}
	}
}

func TestTableIncompleteOrNestedSubscriptsStayLiteral(t *testing.T) {
	f := testFont(t)
	for _, text := range []string{"U_AK/V", "U_{AK", "U_{}", "U_{ }", "U_{A_{K}_{V}}", "{AK}", "实验 I/(10⁻¹¹ A)"} {
		plan := f.tableText(text, 5, 200)
		var chars strings.Builder
		for _, letter := range plan.lines[0] {
			chars.WriteRune(letter.char)
			if letter.size != 5 || letter.shift != 0 {
				t.Fatalf("literal input %q was partially interpreted", text)
			}
		}
		if chars.String() != text {
			t.Fatalf("literal input changed: %q -> %q", text, chars.String())
		}
	}
	if len(f.tableText("U_{A\nK}", 5, 100).lines) != 2 {
		t.Fatal("a marker consumed an explicit newline")
	}
}

func TestTableSubscriptHeightFitsRowsAndExplicitLines(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.FontSize = 5
	table.Cells[0].Text = "U_{AK}/V"
	table.Cells[1].Text = "U_{AK}/V\nI/(10⁻¹¹ A)"
	table.Cells[2].Text = "_{🙂}"
	render := f.renderTable(table, DefaultLayout())
	if render.RowHeights[0] != 10 || render.RowHeights[1] != 10 || len(render.Issues) != 0 {
		t.Fatal("subscripts or explicit lines changed fixed row heights", render.RowHeights, render.Issues)
	}
	for _, cell := range render.Cells {
		for _, stroke := range cell.Strokes {
			for _, p := range stroke {
				if p.Y < cell.Y+table.Padding-1e-8 || p.Y > cell.Y+cell.Height-table.Padding+1e-8 {
					t.Fatal("subscript ink was clipped by the row", p, cell)
				}
			}
		}
	}
	missing := render.Cells[2].Missing
	if len(missing) != 1 || missing[0].Char != "🙂" || math.Abs(missing[0].Size-render.Cells[2].FontSize*.65) > 1e-8 || missing[0].Y < render.Cells[2].Y+table.Padding-1e-8 {
		t.Fatal("missing subscript glyph lost its size or vertical position", missing)
	}
	doc, err := f.LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExportPage(doc, 0, testWriting()); err == nil {
		t.Fatal("missing subscript glyph was silently exported")
	}
}
