package engine

import (
	"math"
	"strings"
	"testing"
)

func TestFormulaLeadingWhitespaceMovesInkAndEveryCaret(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LetterSpacing, s.LetterGap = id, 6, 70, .2
		formula := `\[Δ=2d(\frac{1}{cos i}−\frac{sin^{2}i}{cos i})\]`
		base, err := f.Layout(formula, s)
		if err != nil {
			t.Fatal(err)
		}
		for _, prefix := range []string{" ", "  ", "\t", " \t　"} {
			doc, err := f.Layout(prefix+formula+" \t", s)
			if err != nil {
				t.Fatal(err)
			}
			page := doc.Pages[0]
			wantX := s.MarginLeft
			for i, r := range []rune(prefix) {
				cell := page.Cells[i]
				width := f.advance(r, s)
				if cell.Start != i || cell.End != i+1 || math.Abs(cell.X-wantX) > 1e-8 || math.Abs(cell.Width-width) > 1e-8 {
					t.Fatalf("%s %q: whitespace has no independent editing cell: %+v, want x %g width %g", id, prefix, cell, wantX, width)
				}
				found := false
				for _, caret := range page.Carets {
					found = found || caret.Index == i && math.Abs(caret.X-wantX) < 1e-8
				}
				if !found {
					t.Fatal("whitespace caret is unreachable", prefix, i)
				}
				wantX += width
			}
			if len(page.Formulas) != 1 || math.Abs(page.Formulas[0].X-wantX) > 1e-8 {
				t.Fatal("indent was ignored", prefix, page.Formulas)
			}
			if page.Formulas[0].Start != len([]rune(prefix)) || page.Text != prefix+formula+" \t" || len(page.Issues) > 0 {
				t.Fatal("formula source/range changed", page)
			}
			for i, stroke := range base.Pages[0].Strokes {
				for j, point := range stroke {
					got := page.Strokes[i][j]
					if math.Abs(got.X-point.X-(wantX-s.MarginLeft)) > 1e-8 || math.Abs(got.Y-point.Y) > 1e-8 {
						t.Fatal("indent changed formula size or vertical position")
					}
				}
			}
			if _, err := ExportAll(doc, DefaultWriting()); err != nil {
				t.Fatal(err)
			}
			code, err := ExportPage(doc, 0, DefaultWriting())
			if err != nil {
				t.Fatal(err)
			}
			written := replayWrittenStrokes(t, string(code), Point{Y: s.PaperHeight}, *DefaultWriting().PenDown)
			for i, stroke := range page.Strokes {
				for j, point := range stroke {
					if math.Hypot(point.X-written[i][j].X, s.PaperHeight-point.Y-written[i][j].Y) > .00072 {
						t.Fatal("export did not preserve the formula indent")
					}
				}
			}
		}
	}
}

func TestExcessiveFormulaIndentExplainsHowToRecover(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	for _, prefix := range []string{strings.Repeat("\t", 20), strings.Repeat(" ", 53)} {
		if _, err := f.Layout(prefix+`\[123456789\]`, s); err == nil || !strings.Contains(err.Error(), "缩进") {
			t.Fatal("excessive indentation did not give a specific recovery hint", err)
		}
	}
}

func TestFormulaTabsBetweenEquationsUseBodyWidth(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	a, b := `\(x=1\)`, `\(y=2\)`
	doc, err := f.Layout(a+"\t\t"+b, s)
	if err != nil {
		t.Fatal(err)
	}
	formulas := doc.Pages[0].Formulas
	if len(formulas) != 2 || doc.Pages[0].Cells[0].Y != doc.Pages[0].Cells[3].Y || math.Abs(formulas[1].X-formulas[0].X-formulas[0].Width-2*f.advance('\t', s)) > 1e-8 {
		t.Fatal("formula separators still use a fixed gap", formulas)
	}
}

func TestIndentedWrappedFormulaStaysWithItsWhitespaceAcrossPages(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight, s.LineCount, s.FontSize = 80, 80, 8, 6
	prefix := " \t"
	formula := `\[x` + strings.Repeat(`=\frac{123+456}{2}`, 8) + `\]`
	text := strings.Repeat("正文\n", 7) + prefix + formula
	doc, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	joined, count := "", 0
	indent := f.advance(' ', s) + f.advance('\t', s)
	for _, page := range doc.Pages {
		joined += page.Text
		if strings.Contains(page.Text, prefix) && len(page.Formulas) == 0 {
			t.Fatal("indent was left on the previous page")
		}
		for _, formula := range page.Formulas {
			count++
			if math.Abs(formula.X-s.MarginLeft-indent) > 1e-8 || formula.X+formula.Width > s.PaperWidth-s.MarginRight+1e-8 {
				t.Fatal("wrapped formula lost indent or exceeded the page", formula)
			}
		}
	}
	if joined != text || count < 2 || len(doc.Pages) < 2 {
		t.Fatal("wrapped source or pagination changed", count, len(doc.Pages))
	}
}
