package engine

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

// A single vertical stroke cannot deform. Any displacement therefore proves
// a spacing change, independently of the existing glyph-shape variation.
func spacingFont() *Font {
	return &Font{Units: 10, Glyphs: map[rune]Glyph{'字': {Width: 6, Strokes: []Stroke{{{3, 1}, {3, 6}}}}}}
}

func TestSpacingActuallyMovesLettersAndLines(t *testing.T) {
	f := spacingFont()
	s := DefaultLayout()
	s.FontSize = 6
	s.HandwritingSeed = 42
	text := strings.Repeat("字", 10) + "\n" + strings.Repeat("字", 10) + "\n" + strings.Repeat("字", 10)
	a, err := f.layoutFlow(text, s, nil, 0, false, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	s.HandwritingVariation = 20
	b, err := f.layoutFlow(text, s, nil, 0, false, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := f.layoutFlow(text, s, nil, 0, false, false, 0)
	if !reflect.DeepEqual(b, c) {
		t.Fatal("spacing rerolled")
	}
	p, q := a.Pages[0], b.Pages[0]
	if len(p.Strokes) != len(q.Strokes) || p.Text != q.Text {
		t.Fatal("content changed")
	}
	xChanged, yChanged := false, false
	for i := range p.Strokes {
		dx := q.Strokes[i][0].X - p.Strokes[i][0].X
		dy := q.Strokes[i][0].Y - p.Strokes[i][0].Y
		xChanged = xChanged || math.Abs(dx) > .01
		yChanged = yChanged || math.Abs(dy) > .01
		if math.Abs(dx) > .5 || math.Abs(dy) > .5 {
			t.Fatal("spacing excessive", dx, dy)
		}
		if math.Abs(q.Cells[i].Y-p.Cells[i].Y-dy) > 1e-8 {
			t.Fatal("caret/selection line did not follow ink")
		}
	}
	if !xChanged || !yChanged {
		t.Fatal("font changed without spacing", xChanged, yChanged)
	}
	// Equal characters no longer have equal advances or equal line pitches.
	dx1 := q.Strokes[1][0].X - q.Strokes[0][0].X
	dx2 := q.Strokes[2][0].X - q.Strokes[1][0].X
	dy1 := q.Cells[10].Y - q.Cells[0].Y
	dy2 := q.Cells[20].Y - q.Cells[10].Y
	if math.Abs(dx1-dx2) < .01 || math.Abs(dy1-dy2) < .01 {
		t.Fatal("spacing remains uniform", dx1, dx2, dy1, dy2)
	}
	s.HandwritingVariation = 0
	restored, _ := f.layoutFlow(text, s, nil, 0, false, false, 0)
	if !reflect.DeepEqual(a.Pages, restored.Pages) {
		t.Fatal("disabling spacing did not restore layout")
	}
}

func TestMultilineTableSpacingStaysInsideCell(t *testing.T) {
	f := spacingFont()
	s := DefaultLayout()
	s.HandwritingVariation = 100
	s.HandwritingSeed = 42
	plan := f.tableText("字字字\n字字字\n字字字", 6, 40)
	plain, _ := f.placeTableText(plan, 1, 0, 0, 40, 30, "left", "middle")
	plan.variation = s.variation("multiline")
	varied, _ := f.placeTableText(plan, 1, 0, 0, 40, 30, "left", "middle")
	if reflect.DeepEqual(plain, varied) {
		t.Fatal("table spacing unchanged")
	}
	if math.Abs((varied[3][0].Y-varied[0][0].Y)-(varied[6][0].Y-varied[3][0].Y)) < .01 {
		t.Fatal("table line pitches still identical")
	}
	for _, stroke := range varied {
		for _, p := range stroke {
			if p.X < 1 || p.X > 39 || p.Y < 1 || p.Y > 29 {
				t.Fatal("cell overflow", p)
			}
		}
	}
}

func TestImportedEquationBandsMoveTogetherAndStayBounded(t *testing.T) {
	data, err := os.ReadFile("testdata/hall-imported-equations.json")
	if err != nil {
		t.Fatal(err)
	}
	var table TableAsset
	if err = json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	f := testFont(t)
	s := DefaultLayout()
	s.PaperWidth = 210
	s.PaperHeight = 297
	plain := f.renderTable(table, s)
	for _, amount := range []float64{20, 100} {
		for _, seed := range []uint32{0, 42, 2321509729} {
			moved := f.renderTable(table, s)
			s.HandwritingVariation = amount
			s.HandwritingSeed = seed
			varyTableBands(table, &moved, s)
			s.HandwritingVariation = 0
			changes := false
			for i, cell := range plain.Cells {
				if len(cell.Strokes) == 0 {
					continue
				}
				dy := moved.Cells[i].Strokes[0][0].Y - cell.Strokes[0][0].Y
				changes = changes || math.Abs(dy) > .01
				for j, stroke := range cell.Strokes {
					for k, p := range stroke {
						q := moved.Cells[i].Strokes[j][k]
						if q.X != p.X || math.Abs(q.Y-p.Y-dy) > 1e-8 {
							t.Fatal("formula band did not translate rigidly")
						}
						if q.Y < 0 || q.Y > moved.Height {
							t.Fatal("equation left its frame")
						}
					}
				}
				// Every fraction bar directly under this numerator moves by the same amount.
				if cell.Borders != nil && cell.Borders.Bottom != nil && *cell.Borders.Bottom {
					found := false
					for j, bar := range plain.Strokes[:len(tableLines(table, table.RowHeights))] {
						if math.Abs(bar[0].Y-(cell.Y+cell.Height)) < 1e-8 && bar[0].X >= cell.X-1e-8 && bar[0].X < cell.X+cell.Width {
							if math.Abs(moved.Strokes[j][0].Y-bar[0].Y-dy) > 1e-8 {
								t.Fatal("fraction bar detached")
							}
							found = true
						}
					}
					if !found {
						t.Fatal("fixture fraction bar missing")
					}
				}
			}
			if !changes {
				t.Fatal("equation bands unchanged", amount, seed)
			}
		}
	}
}

func TestSpacingAtPageEdgesAndObjects(t *testing.T) {
	for _, fontID := range []string{FontHandwriting, FontDC} {
		for _, seed := range []uint32{0, 42, 2321509729} {
			s := DefaultLayout()
			s.FontID = fontID
			s.FontSize = 6
			s.HandwritingVariation = 100
			s.HandwritingSeed = seed
			text := strings.Repeat("实验数据 8888，霍尔效应。\n", 24)
			table := TableAsset{ID: "obstacle", Page: 0, X: 30, Y: 30, ColumnWidths: []float64{50}, RowHeights: []float64{20}, Style: "grid", FontSize: 4, Padding: 1, Cells: []TableCell{{Row: 0, Col: 0, RowSpan: 1, ColSpan: 1, Text: "测量"}}}
			doc, err := testFont(t).LayoutWithObjects(text, s, nil, []TableAsset{table})
			if err != nil {
				t.Fatal(fontID, seed, err)
			}
			for _, page := range doc.Pages {
				for _, cell := range page.Cells {
					if cell.Y < s.MarginTop-1e-8 || cell.Y+cell.Height > s.PaperHeight-s.MarginBottom+1e-8 {
						t.Fatal("selection cell outside page", cell)
					}
				}
				if _, err := GenerateGCode(page, s, testWriting()); err != nil {
					t.Fatal(fontID, seed, err)
				}
			}
		}
	}
}
