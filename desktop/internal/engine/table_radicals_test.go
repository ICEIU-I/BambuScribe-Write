package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestTableRadicalSpansMeanBarsAndSubscripts(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, err := BuiltinFont(id)
		if err != nil {
			t.Fatal(err)
		}
		original := f.Glyphs['√']
		for _, source := range []string{"√(d̅_{1}d̅_{2})", "√{d̅_{1}d̅_{2}}", "√（d̅_{1}d̅_{2}）", "√(√(d̅_{1})d̅_{2})"} {
			plan := f.tableText(source, 6, 200)
			line := plan.lines[0]
			if line[0].radical == 0 || line[0].missing {
				t.Fatal("root not interpreted", source)
			}
			root := line[0].glyph.Strokes[1]
			barY := root[len(root)-1].Y / f.Units * line[0].size
			width := 0.0
			for _, l := range line {
				width += l.width
			}
			end := root[len(root)-1].X / f.Units * line[0].size
			if math.Abs(end-width) > 1e-8 || root[len(root)-2].Y != root[len(root)-1].Y {
				t.Fatal("bar does not span its group", source)
			}
			for _, l := range line[1:] {
				for _, stroke := range l.glyph.Strokes {
					for _, p := range stroke {
						if barY >= l.shift+p.Y/f.Units*l.size {
							t.Fatal("root bar collides with radicand", source)
						}
					}
				}
				if strings.ContainsRune("(){}（）", l.char) {
					t.Fatal("scope delimiters printed", source)
				}
			}
			for _, variation := range []float64{0, 100} {
				settings := DefaultLayout()
				settings.HandwritingVariation = variation
				plan.variation = settings.variation("radical")
				ink, missing := f.placeTableText(plan, 0, 0, 0, width, plan.height, "left", "top")
				if len(missing) != 0 {
					t.Fatal(missing)
				}
				for _, s := range ink {
					for _, p := range s {
						if p.X < -1e-8 || p.X > width+1e-8 || p.Y < -1e-8 || p.Y > plan.height+1e-8 {
							t.Fatal("radical escaped fitted cell", p)
						}
					}
				}
			}
		}
		if !reflect.DeepEqual(original, f.Glyphs['√']) {
			t.Fatal("mutated shared glyph")
		}
	}
}

func TestTableIncompleteRadicalsStayLiteral(t *testing.T) {
	f := testFont(t)
	for _, source := range []string{"√", "√x", "√()", "√(d_{1}", "√{x)", "√\n(x)"} {
		plan := f.tableText(source, 5, 200)
		for _, line := range plan.lines {
			for _, l := range line {
				if l.radical > 0 {
					t.Fatal("partial input consumed", source)
				}
			}
		}
	}
}

func TestTableAnswerBlankHasWidthWithoutInventedInk(t *testing.T) {
	f := testFont(t)
	for _, source := range []string{"d̅_{1}=        mm", "d̅_{1}=        "} {
		padded := f.tableText(source, 5, 200)
		compact := f.tableText(strings.ReplaceAll(source, "        ", ""), 5, 200)
		if math.Abs(padded.widths[0]-compact.widths[0]-20) > 1e-8 {
			t.Fatal("answer space collapsed", source)
		}
		ink, missing := f.placeTableText(padded, 0, 0, 0, 200, 30, "left", "top")
		plain, _ := f.placeTableText(compact, 0, 0, 0, 200, 30, "left", "top")
		if len(ink) != len(plain) || len(missing) != 0 {
			t.Fatal("blank introduced ink/missing glyph", source)
		}
	}
}

func TestTextBoxRadicalKeepsFixedGapsInsideItsMeasuredWidth(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		settings := DefaultLayout()
		settings.LetterGap = .7
		settings.LetterSpacing = 140
		for _, source := range []string{"√(d̅_{1}d̅_{2})", "d̅=√(d̅_{1}d̅_{2})=        mm"} {
			width, height := f.MeasureTextBoxText(source, 6, settings)
			plan := f.tableTextWithSpacing(source, 6, 1e9, &settings)
			ink, missing := f.placeTableText(plan, 0, 0, 0, width, height, "left", "top")
			if len(missing) != 0 {
				t.Fatal(missing)
			}
			for _, stroke := range ink {
				for _, p := range stroke {
					if p.X < -1e-7 || p.X > width+1e-7 || p.Y < -1e-7 || p.Y > height+1e-7 {
						t.Fatal("fixed gap escaped measured radical box", p, width, height)
					}
				}
			}
			spacing := f.textBoxSpacing(plan, settings.LetterGap, "left")
			if len(spacing.StrokeGaps) != len(ink) {
				t.Fatal("radical gap mapping lost strokes")
			}
		}
	}
}
