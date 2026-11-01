package engine

import (
	"math"
	"reflect"
	"testing"
)

func TestTableMeanVoltageAttachesBarWithoutExtraWidth(t *testing.T) {
	f := testFont(t)
	base := f.Glyphs['U']
	plain := f.tableText("U_{H}/mV", 8.21, 100)
	for _, text := range []string{"U\u0305_{H}/mV", "U\u0304_{H}/mV"} {
		plan := f.tableText(text, 8.21, 100)
		if len(plan.lines[0]) != len(plain.lines[0]) || plan.widths[0] != plain.widths[0] {
			t.Fatal("bar must not occupy an extra character width", plan)
		}
		ink, missing := f.placeTableText(plan, 0, 0, 0, 100, 20, "left", "top")
		plainInk, _ := f.placeTableText(plain, 0, 0, 0, 100, 20, "left", "top")
		if len(missing) != 0 || len(ink) != len(plainInk)+1 {
			t.Fatal("mean bar missing or extra formula strokes", missing)
		}
		barIndex := len(base.Strokes)
		bar := ink[barIndex]
		if len(bar) != 2 || bar[0].Y != bar[1].Y {
			t.Fatal("mean bar is not horizontal", bar)
		}
		for _, stroke := range ink[:barIndex] {
			for _, p := range stroke {
				if bar[0].Y >= p.Y || p.X < bar[0].X-1e-8 || p.X > bar[1].X+1e-8 {
					t.Fatal("mean bar is not above the U ink", bar, p)
				}
			}
		}
		withoutBar := append(append([]Stroke(nil), ink[:barIndex]...), ink[barIndex+1:]...)
		for i, stroke := range withoutBar {
			for j, p := range stroke {
				q := plainInk[i][j]
				if math.Abs(p.X-q.X) > 1e-8 || math.Abs(p.Y+plan.inkTop-q.Y-plain.inkTop) > 1e-8 {
					t.Fatal("bar changed base letter, subscript or unit shape")
				}
			}
		}
	}
	if !reflect.DeepEqual(f.Glyphs['U'], base) {
		t.Fatal("decorated text changed the shared font")
	}
}

func TestTableOverlineRespectsSubscriptsAndMissingMarks(t *testing.T) {
	f := testFont(t)
	plan := f.tableText("I_{H\u0305}", 5, 100)
	if len(plan.lines[0]) != 2 || plan.lines[0][1].size != 3.25 || plan.lines[0][1].shift != 2 || len(plan.lines[0][1].glyph.Strokes) != len(f.Glyphs['H'].Strokes)+1 {
		t.Fatal("subscript mean bar lost its scale or position")
	}
	for _, text := range []string{"\u0305U", "U \u0305", "U_{\u0305}"} {
		plan := f.tableText(text, 5, 100)
		_, missing := f.placeTableText(plan, 0, 0, 0, 100, 20, "left", "top")
		if len(missing) != 1 || missing[0].Char != "\u0305" {
			t.Fatalf("unattached mark in %q was silently dropped", text)
		}
	}
}

func TestTableWritingBlankIsContinuousAndPreservesAdvance(t *testing.T) {
	f := testFont(t)
	for _, text := range []string{"____", "_{____}"} {
		plan := f.tableText(text, 5, 100)
		ink, missing := f.placeTableText(plan, 0, 0, 0, 100, 20, "left", "top")
		if len(ink) != 1 || len(ink[0]) != 2 || len(missing) != 0 || ink[0][0].Y != ink[0][1].Y {
			t.Fatal("blank is still dashed or lost", text, ink)
		}
		letter := plan.lines[0][0]
		if math.Abs(plan.widths[0]-4*letter.width) > 1e-8 || ink[0][1].X-ink[0][0].X < 3*letter.width {
			t.Fatal("joining the blank shortened its writing space")
		}
	}
	// A literal underscore and two separate writable blanks remain distinct.
	plan := f.tableText("_ ____ ____", 5, 100)
	ink, _ := f.placeTableText(plan, 0, 0, 0, 100, 20, "left", "top")
	if len(ink) != 3 || len(ink[0]) != len(f.Glyphs['_'].Strokes[0]) || ink[1][1].X >= ink[2][0].X {
		t.Fatal("joining blanks changed a literal underscore or crossed a space")
	}
}

func TestTableMeanVoltageAndBlanksFitNativeRows(t *testing.T) {
	f := testFont(t)
	a := sampleTable()
	a.FontSize = 5
	a.Cells[0].Text = "U\u0305_{H}/mV"
	a.Cells[1].Text = "X_{c}=____"
	a.Cells[2].Text = "U\u0305_{H}\nY_{c}=____"
	a.ColumnWidths = []float64{40, 40}
	render := f.renderTable(a, DefaultLayout())
	if len(render.Issues) != 0 {
		t.Fatal(render.Issues)
	}
	for _, c := range render.Cells {
		if len(c.Missing) != 0 {
			t.Fatal(c.Missing)
		}
		for _, stroke := range c.Strokes {
			for _, p := range stroke {
				if p.X < c.X+a.Padding-1e-8 || p.X > c.X+c.Width-a.Padding+1e-8 || p.Y < c.Y+a.Padding-1e-8 || p.Y > c.Y+c.Height-a.Padding+1e-8 {
					t.Fatal("decoration escaped its cell", p, c)
				}
			}
		}
	}
	doc, err := f.LayoutWithObjects("", DefaultLayout(), nil, []TableAsset{a})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ExportPage(doc, 0, testWriting()); err != nil {
		t.Fatal("mean voltage still blocks native export", err)
	}
}
