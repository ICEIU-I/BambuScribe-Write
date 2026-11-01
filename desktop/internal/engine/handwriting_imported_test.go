package engine

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

// This is the actual imported vector formula object from the Hall report,
// including its fraction rules and several repetitions of 126.3.
func TestImportedFormulaVariationVisibleAtSmallSize(t *testing.T) {
	data, err := os.ReadFile("testdata/hall-imported-equations.json")
	if err != nil {
		t.Fatal(err)
	}
	var table TableAsset
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if err := table.Validate(); err != nil {
		t.Fatal(err)
	}
	f := testFont(t)
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight = 210, 297
	s.HandwritingSeed = 2321509729
	plain := f.renderTable(table, s)
	s.HandwritingVariation = 36
	varied := f.renderTable(table, s)
	if len(plain.Issues) > 0 || len(varied.Issues) > 0 {
		t.Fatal(plain.Issues, varied.Issues)
	}
	if !reflect.DeepEqual(plain.RowHeights, varied.RowHeights) {
		t.Fatal("formula rows moved")
	}
	grid := len(tableLines(table, table.RowHeights))
	for i, stroke := range plain.Strokes[:grid] {
		other := varied.Strokes[i]
		if len(other) != 2 || other[0].Y != other[1].Y || other[0].X != stroke[0].X || other[1].X != stroke[1].X {
			t.Fatal("fraction bar deformed")
		}
	}
	if reflect.DeepEqual(plain.Strokes[:grid], varied.Strokes[:grid]) {
		t.Fatal("imported equation line spacing did not change")
	}
	total, count := 0.0, 0
	for i, cell := range plain.Cells {
		for j, stroke := range cell.Strokes {
			for k, p := range stroke {
				q := varied.Cells[i].Strokes[j][k]
				total += math.Hypot(p.X-q.X, p.Y-q.Y)
				count++
			}
		}
	}
	// At 96 dpi, 0.11 mm is over 0.4 pixels. The original 36% setting
	// averaged only 0.03 mm and vanished under the 0.26 mm preview pen.
	if count == 0 || total/float64(count) < .11 {
		t.Fatalf("imported text variation is visually negligible: %.4f mm", total/float64(count))
	}
}
