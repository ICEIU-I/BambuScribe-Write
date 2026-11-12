package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestRomanNumeralsUseTheSelectedHandwritingWithoutDistortion(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		for r := rune(0x2160); r <= 0x217f; r++ {
			g, ok := f.Glyphs[r]
			if !ok || len(g.Strokes) == 0 || g.Width <= 0 || g.Width > f.Units {
				t.Fatalf("%s cannot draw Roman numeral %c", id, r)
			}
			for _, stroke := range g.Strokes {
				for _, p := range stroke {
					if !finite(p.X) || !finite(p.Y) || p.X < 0 || p.Y < 0 || p.X > f.Units || p.Y > f.Units {
						t.Fatalf("%s %c leaves its glyph cell: %v", id, r, p)
					}
				}
			}
		}
		for _, pair := range [][2]rune{{'Ⅰ', 'I'}, {'Ⅴ', 'V'}, {'Ⅹ', 'X'}, {'ⅰ', 'i'}, {'ⅿ', 'm'}} {
			if !reflect.DeepEqual(f.Glyphs[pair[0]], f.Glyphs[pair[1]]) {
				t.Fatal("single-letter numeral must retain the selected letter's strokes", id, pair)
			}
		}
		letter, numeral := f.Glyphs['I'], f.Glyphs['Ⅲ']
		if len(numeral.Strokes) != 3*len(letter.Strokes) {
			t.Fatal("III must retain all three letters")
		}
		a, b, c, d, _ := inkBounds(letter.Strokes)
		previousRight := -1.0
		for i := 0; i < 3; i++ {
			x, y, z, w, _ := inkBounds(numeral.Strokes[i*len(letter.Strokes) : (i+1)*len(letter.Strokes)])
			if math.Abs((z-x)/(w-y)-(c-a)/(d-b)) > 1e-8 || x <= previousRight {
				t.Fatal("III distorted or overlapped its letters", id)
			}
			previousRight = z
		}
	}
}

func TestRomanNumeralsWorkInBodySubscriptsAndFlowedFormulas(t *testing.T) {
	text := "第Ⅰ组、第Ⅱ组、第Ⅲ组\n测量值L_{0,Ⅲ}与L_{5,Ⅲ}。\n" +
		`\(λ_{Ⅰ}=\frac{30-3}{3}=9mm\)　\(λ_{Ⅱ}=9mm\)　\(λ_{Ⅲ}=9mm\)` + "\n" +
		`\(\bar{λ}=\frac{λ_{Ⅰ}+λ_{Ⅱ}+λ_{Ⅲ}}{3}=9mm\)`
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FractionScale, s.HandwritingVariation = id, 85, 15
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(id, err)
		}
		var source strings.Builder
		count := 0
		for _, page := range doc.Pages {
			source.WriteString(page.Text)
			count += len(page.Formulas)
			if len(page.Issues) != 0 || len(page.Missing) != 0 {
				t.Fatal("Roman numerals block preview", page.Issues, page.Missing)
			}
		}
		if source.String() != text || count != 4 {
			t.Fatal("formula source or editing ranges lost")
		}
		assertWritingReplay(t, doc)
	}
}
