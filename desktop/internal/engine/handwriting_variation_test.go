package engine

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestHandwritingVariationBoundsTopologyAndFontSafety(t *testing.T) {
	for _, fontID := range []string{FontHandwriting, FontDC} {
		f, err := testFont(t).ForLayout(LayoutSettings{FontID: fontID})
		if err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(f.Glyphs)
		settings := DefaultLayout()
		for _, amount := range []float64{1, 45, 100} {
			settings.HandwritingVariation = amount
			for _, seed := range []uint32{0, 42, math.MaxUint32} {
				settings.HandwritingSeed = seed
				v := settings.variation("test")
				for _, r := range "霍尔效应一0123456789IlU=+-.,|①⁻¹⁹μ" {
					g := f.Glyphs[r]
					l, right, top, bottom := tableGlyphBounds(g)
					out := v.strokes(g.Strokes, 7)
					if len(out) != len(g.Strokes) {
						t.Fatal("stroke topology changed", r)
					}
					for i, stroke := range out {
						if len(stroke) != len(g.Strokes[i]) {
							t.Fatal("vertices changed", r)
						}
						for j, p := range stroke {
							if !finite(p.X) || !finite(p.Y) || p.X < l || p.X > right || p.Y < top || p.Y > bottom {
								t.Fatal("escaped glyph bounds", r, p)
							}
							q := g.Strokes[i][j]
							if math.Hypot(p.X-q.X, p.Y-q.Y) > .22*math.Hypot(right-l, bottom-top) {
								t.Fatal("variation is excessive", r)
							}
						}
					}
				}
			}
		}
		after, _ := json.Marshal(f.Glyphs)
		if string(before) != string(after) {
			t.Fatal("shared font mutated")
		}
	}
}

func TestHandwritingVariationDeterminismOccurrencesAndExport(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	text := "8888\n" + `\(\frac{88}{88}=88\)` + "\f8888"
	base, err := f.LayoutWithObjects(text, s, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.HandwritingSeed = 123
	off, _ := f.LayoutWithObjects(text, s, nil, nil)
	for i := range base.Pages {
		if !reflect.DeepEqual(base.Pages[i].Strokes, off.Pages[i].Strokes) {
			t.Fatal("off changed ink")
		}
	}
	s.HandwritingVariation = 45
	render := func() *Document {
		d, err := f.LayoutWithObjects(text, s, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	a, b := render(), render()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("render not deterministic")
	}
	if len(a.Pages) != len(base.Pages) {
		t.Fatal("pagination changed")
	}
	for i, p := range a.Pages {
		old := base.Pages[i]
		if len(p.Cells) != len(old.Cells) || len(p.Formulas) != len(old.Formulas) || p.Text != old.Text {
			t.Fatal("content or glyph identities changed")
		}
		if reflect.DeepEqual(p.Strokes, old.Strokes) {
			t.Fatal("variation not applied")
		}
		code, err := GenerateGCode(p, s, testWriting())
		if err != nil {
			t.Fatal(err)
		}
		moves := xyMoves(t, code, Point{})
		var expected []Point
		for _, stroke := range p.Strokes {
			if len(stroke) > 1 {
				expected = append(expected, stroke...)
			}
		}
		if len(moves) != len(expected)+1 {
			t.Fatal("export lost stroke points")
		}
		for j, point := range expected {
			if math.Abs(moves[j].X-point.X) > .00051 || math.Abs(moves[j].Y-(s.PaperHeight-point.Y)) > .00051 {
				t.Fatal("preview/export mismatch", j, moves[j], point)
			}
		}
	}
	local := func(layer InkLayer) []Stroke {
		p := layer.Strokes[0][0]
		return appendMathStrokes(nil, layer.Strokes, -p.X, -p.Y, 1)
	}
	if reflect.DeepEqual(local(a.Pages[0].Ink[0]), local(a.Pages[0].Ink[1])) || reflect.DeepEqual(local(a.Pages[0].Ink[0]), local(a.Pages[1].Ink[0])) {
		t.Fatal("repeated characters reused shapes")
	}
	s.HandwritingSeed++
	c := render()
	if reflect.DeepEqual(a.Pages[0].Strokes, c.Pages[0].Strokes) {
		t.Fatal("new seed did not change ink")
	}
	// Index-based erasures survive because all stroke/vertex identities remain.
	layer := a.Pages[0].Ink[0]
	cuts := []InkCut{{Part: layer.Part, Source: layer.Source, Stroke: 0, Ranges: [][2]float64{{0, float64(len(layer.Strokes[0]) - 1)}}}}
	cut := cutLayer(c.Pages[0].Ink[0], []PageErasure{{Cuts: &cuts}})
	if !reflect.DeepEqual(cut, c.Pages[0].Ink[0].Strokes[1:]) {
		t.Fatal("erased stroke restored after reseed")
	}
}

func TestHandwritingVariationKeepsFormulaAndTableRules(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	s.HandwritingVariation = 100
	v := s.variation("formula")
	source := `\bar{U}_{H}=\frac{88}{88}`
	plain, _ := f.formulaBox(source, s)
	varied, _ := f.formulaBox(source, s, v)
	if math.Abs(plain.width-varied.width) > s.FontSize || plain.top != varied.top || plain.bottom != varied.bottom {
		t.Fatal("formula spacing escaped its limit")
	}
	if reflect.DeepEqual(plain.strokes, varied.strokes) {
		t.Fatal("formula unchanged")
	}
	for i, stroke := range plain.strokes {
		if len(stroke) == 2 && stroke[0].Y == stroke[1].Y && (len(varied.strokes[i]) != 2 || varied.strokes[i][0].Y != varied.strokes[i][1].Y) {
			t.Fatal("formula rule changed")
		}
	}
	for _, kind := range []string{"", "textbox"} {
		table := sampleTable()
		table.Kind = kind
		for i := range table.Cells {
			table.Cells[i].Text = "8̅8____"
		}
		if kind == "textbox" {
			table.Style = "none"
			table.ColumnWidths = []float64{60}
			table.RowHeights = []float64{20}
			table.Cells = table.Cells[:1]
		}
		s.HandwritingVariation = 0
		a := f.renderTable(table, s)
		s.HandwritingVariation = 100
		b := f.renderTable(table, s)
		grid := len(tableLines(table, a.RowHeights))
		if !reflect.DeepEqual(a.Strokes[:grid], b.Strokes[:grid]) || a.Width != b.Width || a.Height != b.Height {
			t.Fatal("table geometry changed")
		}
		if reflect.DeepEqual(a.Cells[0].Strokes, b.Cells[0].Strokes) {
			t.Fatal("cell text unchanged", kind)
		}
		for i, stroke := range a.Cells[0].Strokes {
			if len(stroke) == 2 && stroke[0].Y == stroke[1].Y && (len(b.Cells[0].Strokes[i]) != 2 || b.Cells[0].Strokes[i][0].Y != b.Cells[0].Strokes[i][1].Y || math.Abs((stroke[1].X-stroke[0].X)-(b.Cells[0].Strokes[i][1].X-b.Cells[0].Strokes[i][0].X)) > 1e-8) {
				t.Fatal("overline or blank rule changed")
			}
		}
	}
	plan := f.tableText("𠮷̅", 6, 100)
	plan.variation = v
	_, missing := f.placeTableText(plan, 0, 0, 0, 100, 30, "left", "top")
	if len(missing) != 1 {
		t.Fatal("unsupported decorated character must report a missing glyph")
	}
}

func TestHandwritingVariationFormulaWrappingKeepsEachShape(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize, s.HandwritingVariation, s.HandwritingSeed = 6, 100, 42
	source := `\frac{88}{88}=88=88=88=88`
	v := s.variation("wrapping")
	whole, err := f.formulaBox(source, s, v)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := f.formulaLines(source, s, whole.width*.6, v)
	if err != nil || len(lines) < 2 {
		t.Fatal("formula did not wrap", err)
	}
	index := 0
	for _, line := range lines {
		start := whole.strokes[index][0]
		first := line.box.strokes[0][0]
		for _, stroke := range line.box.strokes {
			for j, p := range stroke {
				q := whole.strokes[index][j]
				if math.Abs((p.X-first.X)-(q.X-start.X)) > 1e-8 || math.Abs((p.Y-first.Y)-(q.Y-start.Y)) > 1e-8 {
					t.Fatal("wrapping rerolled a character")
				}
			}
			index++
		}
	}
}

func TestHandwritingVariationImageTextAndValidation(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	asset := ImageAsset{ID: "label", X: 10, Y: 10, Width: 90, Height: 60}
	for _, text := range []string{"888霍尔效应", "U_{H}=88", `\frac{88}{88}`} {
		box := OCRBox{Text: text, X: .1, Y: .1, W: .7, H: .3}
		if strings.ContainsAny(text, "_\\") {
			box.TextFormat = "math"
		}
		s.HandwritingVariation = 0
		a := imageRegionTextStrokes(f, box, asset, s, 900, 600)
		s.HandwritingVariation = 45
		b := imageRegionTextStrokes(f, box, asset, s, 900, 600)
		if len(a) == 0 || len(a) != len(b) || reflect.DeepEqual(a, b) {
			t.Fatal("image text variation failed", text)
		}
		for _, stroke := range b {
			for _, p := range stroke {
				if p.X < 19-1e-8 || p.X > 82+1e-8 || p.Y < 16-1e-8 || p.Y > 34+1e-8 {
					t.Fatal("image text outside region", p)
				}
			}
		}
	}
	for _, value := range []float64{-1, 101, math.Inf(1), math.NaN()} {
		s.HandwritingVariation = value
		if _, err := f.Layout("测试", s); err == nil {
			t.Fatal("invalid variation accepted", value)
		}
	}
}
