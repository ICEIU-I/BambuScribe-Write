package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestBodyRadicalFitsContentWithoutMovingItsBaseline(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		for _, spacing := range []float64{70, 150} {
			s := DefaultLayout()
			s.FontID, s.FontSize, s.LetterSpacing, s.LetterGap = id, 6, spacing, .7
			var previousWidth float64
			for _, content := range []string{"a", "a+b", "a_{1}+b_{2}"} {
				source := "√(" + content + ")"
				font, runes := f.bodyRunes(source, s)
				if len(runes) != 1 || font.bodySource(runes[0]) != source {
					t.Fatalf("root was not grouped: %s %q", id, source)
				}
				g := font.Glyphs[runes[0]]
				plain, letters := f.bodyRunes(content, s)
				page := Page{}
				index := 0
				plain.appendTextFragment(&page, letters, textFragment{textSpan: textSpan{0, 200}, used: 100}, s, 0, &index, map[rune]bool{}, &Document{}, handwritingVariation{})
				if len(g.Strokes) != len(page.Strokes)+2 {
					t.Fatal("root printed its scope delimiters or lost content", source)
				}
				bar := g.Strokes[1]
				if len(bar) != 2 || bar[0].Y != bar[1].Y || bar[1].X <= previousWidth {
					t.Fatal("root bar did not grow with its content", source, bar)
				}
				previousWidth = bar[1].X
				scale := s.FontSize / f.Units
				dx := g.Strokes[2][0].X*scale - page.Strokes[0][0].X
				for i, stroke := range page.Strokes {
					for j, p := range stroke {
						q := g.Strokes[i+2][j]
						if math.Abs(q.X*scale-dx-p.X) > 1e-8 || math.Abs(q.Y*scale-p.Y) > 1e-8 {
							t.Fatal("root resized or vertically moved its contents", id, source)
						}
						if q.X > bar[1].X+1e-8 || q.Y <= bar[0].Y {
							t.Fatal("root bar did not cover the complete content", source, q, bar)
						}
					}
				}
			}
		}
	}
}

func TestBodyRadicalWrappingSourceAndExport(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize, s.PaperWidth, s.PaperHeight, s.LineCount = 5, 55, 45, 5
	source := strings.Repeat("正文√(a_{1}+b²)，", 18) + "\f后续√（a+b）"
	doc, err := f.Layout(source, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) < 2 || len(doc.Missing) != 0 {
		t.Fatal("root pagination failed")
	}
	joined, count := "", 0
	encoded := utf16.Encode([]rune(source))
	for _, page := range doc.Pages {
		joined += page.Text
		if len(page.Formulas) != 0 || len(page.Issues) != 0 {
			t.Fatal("inline root changed into a display formula or overlapped", page.Issues)
		}
		for _, cell := range page.Cells {
			text := string(utf16.Decode(encoded[cell.Start:cell.End]))
			if strings.HasPrefix(text, "√") {
				count++
				if !strings.HasSuffix(text, ")") && !strings.HasSuffix(text, "）") {
					t.Fatal("root split across lines", text)
				}
			}
		}
	}
	if joined != source || count != 19 {
		t.Fatal("root source text/selection lost", count)
	}
	if _, err := ExportAll(doc, DefaultWriting()); err != nil {
		t.Fatal(err)
	}
	font := *f
	font.Glyphs = map[rune]Glyph{'𠀀': f.Glyphs['中'], '√': f.Glyphs['√'], 'a': f.Glyphs['a']}
	doc, err = font.Layout("√(𠀀)a", s)
	if err != nil || len(doc.Pages[0].Cells) != 2 || doc.Pages[0].Cells[0].End != 5 || doc.Pages[0].Cells[1].Start != 5 {
		t.Fatal("root lost UTF-16 caret indices", err)
	}
}

func TestBodyRadicalInvalidInputAndFontIsolation(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	before, _ := f.Layout("正文√2及R_{g}", s)
	for _, source := range []string{"√", "√2", "√()", "√( )", "√(a", "√(a）", "√(R_{})", "√(🧪)", strings.Repeat("√(", 9) + "a" + strings.Repeat(")", 9)} {
		font, runes := f.bodyRunes(source, s)
		if string(runes) != source || len(font.bodyScripts) != 0 {
			t.Fatal("invalid root did not remain literal", source)
		}
	}
	f.bodyRunes("√(a_{1}+√(b²))", s)
	after, _ := f.Layout("正文√2及R_{g}", s)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("root modified the shared font or existing layout")
	}
	doc, err := f.Layout("√(🧪)", s)
	if err != nil || len(doc.Missing) == 0 {
		t.Fatal("missing root content was silently dropped", err)
	}
	if _, err := ExportAll(doc, DefaultWriting()); err == nil {
		t.Fatal("missing root content was exported")
	}
	s.FontSize, s.PaperWidth = 10, 40
	if _, err := f.Layout("√("+strings.Repeat("a+", 30)+"b)", s); err == nil {
		t.Fatal("oversize root was split or exported beyond the page")
	}
}

func TestBodyNestedRadicalBoundsAndHandwriting(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 6, 16
		s.MarginTop, s.HandwritingVariation = 0, 100
		source := "√(a_{1}+√(b²))"
		_, runes := f.bodyRunes(source, s)
		if len(runes) != 1 {
			t.Fatal("nested root was not grouped")
		}
		doc, err := f.Layout(source, s)
		if err != nil {
			t.Fatal(err)
		}
		strokes := doc.Pages[0].Strokes
		if strokes[1][0].Y != strokes[1][1].Y {
			t.Fatal("handwriting tilted the radical bar")
		}
		// Repeated roots reaching above the normal glyph cell remain on paper.
		tall := strings.Repeat("√(", 8) + "a" + strings.Repeat(")", 8)
		doc, err = f.Layout(tall, s)
		if err != nil {
			t.Fatal(err)
		}
		for _, stroke := range doc.Pages[0].Strokes {
			for _, p := range stroke {
				if p.Y < -1e-8 {
					t.Fatal("root left the top edge", p)
				}
			}
		}
		if _, err := ExportAll(doc, DefaultWriting()); err != nil {
			t.Fatal(err)
		}
		s.HandwritingVariation = 0
		plain, err := f.Layout(source, s)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(strokes, plain.Pages[0].Strokes) {
			t.Fatal("root contents did not receive handwriting variation")
		}
		if strokes[1][1].X-strokes[1][0].X != plain.Pages[0].Strokes[1][1].X-plain.Pages[0].Strokes[1][0].X {
			t.Fatal("handwriting changed root length")
		}
	}
}
