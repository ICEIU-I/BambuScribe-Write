package engine

import (
	"math"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestShortEquationsFlowAcrossRowsAndPagesWithoutLosingEditingRanges(t *testing.T) {
	formula := `\(λ_{1}=\frac{115.10-87.70}{3}=9.133mm\)`
	text := "计算\n" + strings.TrimSuffix(strings.Repeat(formula+"　", 100), "　")
	source := utf16.Encode([]rune(text))
	for _, id := range []string{FontHandwriting, FontDC} {
		for _, fractionScale := range []float64{50, 85, 100} {
			f, _ := BuiltinFont(id)
			s := DefaultLayout()
			s.FontID, s.FontSize, s.LineCount = id, 8.3, 30
			s.PaperWidth, s.PaperHeight = 210, 297
			s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 24, 24, 17, 17
			s.FractionScale, s.HandwritingVariation = fractionScale, 15
			s.PageLayouts = []PageLayout{{LineCount: 30}, {LineCount: 22}}
			doc, err := f.Layout(text, s)
			if err != nil {
				t.Fatal(id, fractionScale, err)
			}
			joined, count, sharedRows := "", 0, 0
			for _, page := range doc.Pages {
				joined += page.Text
				if len(page.Issues) != 0 || len(page.Missing) != 0 {
					t.Fatal("flow caused layout issues", page.Issues, page.Missing)
				}
				for i, item := range page.Formulas {
					count++
					if string(utf16.Decode(source[item.Start:item.End])) != formula {
						t.Fatal("formula source or UTF-16 range changed", item)
					}
					if item.X < s.MarginLeft-1e-8 || item.X+item.Width > s.PaperWidth-s.MarginRight+1e-8 || item.Y+item.Height > s.PaperHeight-s.MarginBottom+1e-8 || item.Height > page.LineHeight {
						t.Fatal("formula exceeds writing area", item)
					}
					if i > 0 && math.Abs(item.Y-page.Formulas[i-1].Y) < .01 {
						sharedRows++
						previous := page.Formulas[i-1]
						if item.X-previous.X-previous.Width < 4-1e-8 {
							t.Fatal("adjacent formulas overlap or have no gap")
						}
					}
				}
			}
			if joined != text || count != 100 || sharedRows == 0 || len(doc.Pages) > 3 {
				t.Fatal("formula flow lost source or failed to compact", count, sharedRows, len(doc.Pages))
			}
			if _, err := ExportAll(doc, DefaultWriting()); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestJoinedFormulaDelimitersAndLongEqualityWrapping(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	short := `\(x=\frac{1}{2}\)`
	long := `\(y` + strings.Repeat(`=\frac{123.45+678.90}{2}`, 12) + `\)`
	for _, text := range []string{short + "　" + short, short + long + " " + short, " " + short + "　" + short, short + "　", short + "　" + long + " "} {
		if _, ok := formulaSource(text); ok {
			t.Fatal("separate equations misread as a single formula")
		}
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal("joined delimiters caused a parser error", err)
		}
		joined := ""
		for _, page := range doc.Pages {
			joined += page.Text
			if len(page.Issues) > 0 {
				t.Fatal(page.Issues)
			}
		}
		if joined != text {
			t.Fatal("wrapped equality changed the source")
		}
	}
}

func TestFormulaFlowAvoidsObjectsAndPreservesTrailingWhitespace(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	text := `\(x=\frac{1}{2}\)　\(y=\frac{2}{3}\) `
	object := RegionBounds{s.MarginLeft + 15, s.MarginTop, 10, 20}
	doc, err := f.layoutFlow(text, s, map[int][]RegionBounds{0: {object}}, 0, false, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Text != text || len(doc.Pages[0].Formulas) != 2 {
		t.Fatal("source or formula objects lost")
	}
	for _, formula := range doc.Pages[0].Formulas {
		if formula.Y < object.Y+object.Height+objectLayoutGap {
			t.Fatal("formula flow overlaps the reserved object band", formula)
		}
	}
}
