package engine

import (
	"math"
	"reflect"
	"testing"
)

func TestTextBoxFitsCaptionAndScalesBothDirections(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		t.Run(id, func(t *testing.T) {
			f, err := BuiltinFont(id)
			if err != nil {
				t.Fatal(err)
			}
			box := sampleTextBox()
			box.Cells[0].Text = "图一 霍尔效应原理图"
			box.FontSize = 8.206
			base := f.renderTable(box, DefaultLayout())
			if len(base.Issues) != 0 || base.FontSize >= box.FontSize {
				t.Fatal("caption did not shrink into its frame", base.FontSize, base.Issues)
			}
			assertMaximumTableFont(t, f, box, base)
			for _, factor := range []float64{.5, 1.5, 2} {
				scaled := box
				scaled.ColumnWidths = []float64{box.ColumnWidths[0] * factor}
				scaled.RowHeights = []float64{box.RowHeights[0] * factor}
				scaled.Padding *= factor
				render := f.renderTable(scaled, DefaultLayout())
				if math.Abs(render.FontSize-base.FontSize*factor) > 1e-7 {
					t.Fatal("font did not scale with frame", factor, render.FontSize)
				}
				assertMaximumTableFont(t, f, scaled, render)
			}
			box.FontSize = 1
			if !reflect.DeepEqual(base, f.renderTable(box, DefaultLayout())) {
				t.Fatal("stored font overrides automatic fitting")
			}
			box.Cells[0].Text = "U̅_{H}\n平均值"
			render := f.renderTable(box, DefaultLayout())
			assertMaximumTableFont(t, f, box, render)
			if !reflect.DeepEqual(render.RowHeights, box.RowHeights) {
				t.Fatal("multiline caption grew instead of fitting")
			}
		})
	}
}
