package engine

import (
	"math"
	"reflect"
	"testing"
)

func TestTextBoxSpacingMatchesBody(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, err := BuiltinFont(id)
		if err != nil {
			t.Fatal(err)
		}
		for _, spacing := range []float64{0, 50, 76.5, 100, 150, 200} {
			for _, gap := range []float64{0, .8} {
				s := DefaultLayout()
				s.FontID, s.LetterSpacing, s.LetterGap = id, spacing, gap
				box := sampleTextBox()
				box.Page, box.X, box.Y = 0, 0, 50
				box.Align, box.VerticalAlign = "left", "top"
				box.Cells[0].Text = "图一 霍尔\tU=2.5μA　原理"
				render := f.renderTable(box, s)
				if len(render.Issues) > 0 {
					t.Fatal(id, spacing, gap, render.Issues)
				}
				s.FontSize = render.FontSize
				body, err := f.Layout(box.Cells[0].Text, s)
				if err != nil {
					t.Fatal(err)
				}
				a, b := render.Cells[0].Strokes, body.Pages[0].Strokes
				if len(a) != len(b) {
					t.Fatal("text box changed body glyphs")
				}
				dx, dy := a[0][0].X-b[0][0].X, a[0][0].Y-b[0][0].Y
				for i, stroke := range a {
					for j, p := range stroke {
						if math.Abs(p.X-b[i][j].X-dx) > 1e-8 || math.Abs(p.Y-b[i][j].Y-dy) > 1e-8 {
							t.Fatalf("%s %.1f%% gap %.1f: caption and body spacing disagree", id, spacing, gap)
						}
					}
				}
				for _, stroke := range a {
					for _, p := range stroke {
						if p.X < box.Padding-1e-7 || p.X > render.Width-box.Padding+1e-7 || p.Y < box.Padding-1e-7 || p.Y > render.Height-box.Padding+1e-7 {
							t.Fatal("caption ink leaves its frame", id, spacing, gap, p)
						}
					}
				}
			}
		}
	}
}

func TestTextBoxSpacingRefitsCaptionsWithoutChangingTables(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	box := sampleTextBox()
	box.ColumnWidths, box.RowHeights = []float64{60}, []float64{30}
	box.Cells[0].Text = "图一 霍尔效应原理图\nU̅_{H}=____"
	base := f.renderTable(box, s)
	table := f.renderTable(sampleTable(), s)
	s.LetterSpacing, s.LetterGap = 76.5, .5
	compact := f.renderTable(box, s)
	if compact.FontSize <= base.FontSize || reflect.DeepEqual(base.Strokes, compact.Strokes) {
		t.Fatal("body spacing did not update the existing caption")
	}
	if !reflect.DeepEqual(table, f.renderTable(sampleTable(), s)) {
		t.Fatal("caption spacing unexpectedly changed table cells")
	}
	for _, render := range []TablePlacement{base, compact} {
		setting := DefaultLayout()
		if render.FontSize == compact.FontSize {
			setting = s
		}
		plan := f.objectText(box, box.Cells[0].Text, render.FontSize+.001, render.Width-2*box.Padding, setting)
		if plan.issue == "" && plan.height <= render.Height-2*box.Padding {
			t.Fatal("caption could use a larger font")
		}
	}
}
