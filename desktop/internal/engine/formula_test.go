package engine

import (
	"math"
	"strings"
	"testing"
)

func TestFractionNominalSizeAndCompactBar(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, err := testFont(t).ForLayout(LayoutSettings{FontID: id})
		if err != nil {
			t.Fatal(err)
		}
		s := DefaultLayout()
		s.FontSize = 6
		s.FractionScale = 85
		base, err := f.formulaBox("4", s)
		if err != nil {
			t.Fatal(err)
		}
		fraction, err := f.formulaBox(`\frac{4}{4}`, s)
		if err != nil {
			t.Fatal(err)
		}
		n := len(base.strokes)
		if len(fraction.strokes) != 2*n+1 {
			t.Fatal("unexpected fraction geometry")
		}
		bounds := func(strokes []Stroke) (float64, float64) {
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, st := range strokes {
				for _, p := range st {
					lo = math.Min(lo, p.Y)
					hi = math.Max(hi, p.Y)
				}
			}
			return lo, hi
		}
		a, b := bounds(base.strokes)
		c, d := bounds(fraction.strokes[:n])
		e, g := bounds(fraction.strokes[n : 2*n])
		if math.Abs((d-c)/(b-a)-.85) > 1e-8 || math.Abs((g-e)/(b-a)-.85) > 1e-8 {
			t.Fatal("fraction changed glyph scale", id)
		}
		if math.Abs(fraction.width-(base.width*.85+6*.16)) > 1e-8 {
			t.Fatal("fraction bar padded beyond content")
		}
		bar := fraction.strokes[len(fraction.strokes)-1][0].Y
		if !(d < bar && bar < e) {
			t.Fatal("numerator, bar and denominator overlap")
		}
	}
}
func TestBodyFormulasFlowAroundObjectsAndStayWhole(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	s.MarginTop = 8
	s.MarginBottom = 8
	s.MarginLeft = 8
	s.MarginRight = 8
	s.LineCount = 24
	formula := `\(U̅_{H}=\frac{|U_{H1}|+|U_{H2}|+|U_{H3}|+|U_{H4}|}{4}=\frac{|25.3|+|-25.5|+|-25.4|+|25.4|}{4}=25.4mV\)`
	text := "数据处理\n" + strings.Repeat(formula+"\n", 35) + "结束"
	doc, err := f.LayoutWithObjects(text, s, nil, []TableAsset{sampleTable()})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Missing) > 0 || len(doc.Pages) < 2 {
		t.Fatal("invalid pagination")
	}
	joined := ""
	count := 0
	for _, p := range doc.Pages {
		joined += p.Text
		if strings.Count(p.Text, `\(`) != strings.Count(p.Text, `\)`) {
			t.Fatal("equation source split across pages")
		}
		for _, m := range p.Formulas {
			count++
			if m.Y+m.Height > s.PaperHeight-s.MarginBottom+1e-8 {
				t.Fatal("formula exceeds page")
			}
			if p.Number == 1 && tableOverlaps(m.X, m.Y, m.Width, m.Height, 12, 20, 60, 20) {
				t.Fatal("formula crosses table")
			}
		}
	}
	if joined != text || count < 35 {
		t.Fatal("formula content lost")
	}
}
func TestFormulaValidationAndProseCompatibility(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 5
	for _, source := range []string{`x=\sqrt{4}=2`, `x=(2)^{3}=8`, `\bar{U}_{H}=\frac{\frac{1}{2}}{4}`, `x=1×10^{-3}`} {
		if _, err := f.Layout(`\(`+source+`\)`, s); err != nil {
			t.Fatal(source, err)
		}
	}
	for _, source := range []string{`\frac{1}{}`, `\frac{1}{2`, `\unknown{1}`, `x_{}`, `\frac{1}{2}}`} {
		if _, err := f.Layout(`\(`+source+`\)`, s); err == nil {
			t.Fatal("invalid formula accepted", source)
		}
	}
	s.FractionScale = 40
	if _, err := f.Layout("正文", s); err == nil {
		t.Fatal("invalid scale accepted")
	}
}

func TestBodyFormulasUseCompactVerticalSpacing(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 5
	s.LineCount = 8
	formula := `\(U=\frac{25.3+25.4}{2}=25.35\)`
	doc, err := f.Layout(formula+"\n"+formula, s)
	if err != nil {
		t.Fatal(err)
	}
	formulas := doc.Pages[0].Formulas
	if len(formulas) != 2 {
		t.Fatal("equations did not fit together")
	}
	gap := formulas[1].Y - formulas[0].Y - formulas[0].Height
	if gap < 0 || gap > s.FontSize*.36+s.ParagraphGap {
		t.Fatalf("excessive formula gap: %g", gap)
	}
}
