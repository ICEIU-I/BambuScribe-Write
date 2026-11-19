package engine

import (
	"math"
	"strings"
	"testing"
)

func readableFormulaSettings() LayoutSettings {
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight = 210, 297
	s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 24, 24, 17, 17
	s.FontSize, s.LineCount, s.LineSpacing, s.FractionScale = 8.3, 30, 85, 100
	return s
}

func TestLargeFractionUsesTwoRowsWithoutShrinkingShortNeighbours(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := readableFormulaSettings()
		s.FontID = id
		fraction := `η=\frac{(ρ-ρ_{0})gd^{2}}{18v(1+\frac{2.4d}{D})}`
		text := `\(` + fraction + `\)　\(x=1\)` + "\n结束"
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(err)
		}
		page := doc.Pages[0]
		if page.Text != text || len(page.Formulas) != 2 {
			t.Fatal("source changed")
		}
		a, b := page.Formulas[0], page.Formulas[1]
		if page.Cells[0].Height != 2*page.LineHeight || page.Cells[len(page.Cells)-1].Y != s.MarginTop+2*page.LineHeight {
			t.Fatal("fraction did not occupy exactly two rules")
		}
		if a.Height < 1.7*page.LineHeight {
			t.Fatal("large fraction still compressed into a single rule", a)
		}
		raw, _ := f.formulaBox("x=1", s)
		if math.Abs(b.Width-raw.width) > 1e-8 {
			t.Fatal("short neighbour was shrunk")
		}
		if b.X < a.X+a.Width+4 || b.Y < a.Y || b.Y+b.Height > a.Y+2*page.LineHeight {
			t.Fatal("neighbour not separated/centred", a, b)
		}
	}
}

func TestTwoRowFractionMovesPastLowerHalfObstaclesAndPageEnd(t *testing.T) {
	f := testFont(t)
	s := readableFormulaSettings()
	equation := `\(η=\frac{(ρ-ρ_{0})gd^{2}}{18v(1+\frac{2.4d}{D})}\)`
	doc, _ := f.Layout("字", s)
	pitch := doc.Pages[0].LineHeight
	count := doc.Pages[0].LineCount
	obstacle := RegionBounds{X: 17, Y: s.MarginTop + pitch + 1, Width: 100, Height: 2}
	doc, err := f.layoutFlow(equation, s, map[int][]RegionBounds{0: {obstacle}}, 0, false, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Formulas[0].Y < obstacle.Y+obstacle.Height+objectLayoutGap {
		t.Fatal("second row overlaps object")
	}
	text := strings.Repeat("字\n", count-1) + equation
	doc, err = f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 2 || len(doc.Pages[0].Formulas) != 0 || len(doc.Pages[1].Formulas) != 1 || doc.Pages[1].Cells[0].Y != s.MarginTop {
		t.Fatalf("two-row fraction stranded on last rule: pages %d, first formulas %v, second cells %v", len(doc.Pages), doc.Pages[0].Formulas, doc.Pages[len(doc.Pages)-1].Cells)
	}
	if doc.Pages[0].Text+doc.Pages[1].Text != text {
		t.Fatal("page break changed source")
	}
}

func TestLongUncertaintyTermsWrapAtEqualsBeforeBoundedWidthFit(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := readableFormulaSettings()
		s.FontID = id
		// Synthetic values exercise a long squared sum without storing an experiment.
		term := `\frac{2.78}{\sqrt{5}}×\sqrt{\frac{(24.50-24.26)^{2}+(23.96-24.26)^{2}+(24.33-24.26)^{2}+(23.88-24.26)^{2}+(24.62-24.26)^{2}}{4}}`
		source := `U_{A}(t)=` + term + `=0.406s`
		text := `\(` + source + `\)`
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(id, err)
		}
		page := doc.Pages[0]
		if page.Text != text || len(page.Formulas) < 2 {
			t.Fatal("long chain was squeezed into one line")
		}
		for _, item := range page.Formulas {
			if item.Width > 176+1e-8 || item.Height > 2*page.LineHeight {
				t.Fatal("term exceeds page")
			}
			if strings.Contains(item.Source, `\frac`) && item.Height < 1.4*page.LineHeight {
				t.Fatal("long term is unreadably small", id, item, page.LineHeight)
			}
		}
		if _, err := ExportAll(doc, DefaultWriting()); err != nil {
			t.Fatal(err)
		}
		tooWide := `\(x=\frac{` + strings.Repeat("123456789+", 30) + `1}{2}\)`
		if _, err := f.Layout(tooWide, s); err == nil || !strings.Contains(err.Error(), "拆分") {
			t.Fatal("unbounded shrink accepted", err)
		}
	}
}
