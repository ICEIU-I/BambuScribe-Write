package engine

import (
	"math"
	"testing"
)

func TestTableNumbersFillRealCellHeightWithoutDistortion(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		t.Run(id, func(t *testing.T) {
			f, err := BuiltinFont(id)
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range []string{"100", "1000", "0.00", "48.00"} {
				a := sampleTable()
				a.ColumnWidths = []float64{22, 28}
				a.RowHeights = []float64{6.78, 6.78}
				a.Cells[0].Text = "很长的表头 +I_{M}, +I_{S}"
				a.Cells[2].Text = text
				render := f.renderTable(a, DefaultLayout())
				cell := render.Cells[2]
				left, right, top, bottom := tableGlyphBounds(Glyph{Strokes: cell.Strokes})
				if bottom-top < 4.77 || math.Abs((top+bottom)/2-(cell.Y+cell.Height/2)) > 1e-7 {
					t.Fatalf("%s ink height %.3f mm is too small or not centered", text, bottom-top)
				}
				unit := f.tableText(text, 1, 100)
				ink, _ := f.placeTableText(unit, 0, 0, 0, 100, 100, "left", "top")
				l, r, u, b := tableGlyphBounds(Glyph{Strokes: ink})
				if math.Abs((right-left)/(bottom-top)-(r-l)/(b-u)) > 1e-7 {
					t.Fatal("cell fit stretched the font")
				}
				a.Cells[0].Text = "短"
				other := f.renderTable(a, DefaultLayout())
				if other.Cells[2].FontSize != cell.FontSize {
					t.Fatal("heading changed the data's size")
				}
				assertMaximumTableFont(t, f, a, other)
			}
		})
	}
}
