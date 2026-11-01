package engine

import (
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestWritingRulesExamplesRenderAndExport(t *testing.T) {
	data, err := os.ReadFile("../../docs/writing-syntax.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 5, 24
		for _, block := range strings.Split(string(data), "```text\n")[1:] {
			source := strings.SplitN(block, "```", 2)[0]
			doc, err := f.Layout(source, s)
			if err != nil {
				t.Fatal(id, source, err)
			}
			if _, err := ExportAll(doc, DefaultWriting()); err != nil {
				t.Fatal(id, source, err)
			}
		}
	}
}

func TestBodyMathCombinationsAndDisplayDelimiters(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 6, 24
		for _, expression := range []string{`x_{i}^{2}`, `x^{2}_{i}`, `(a+b)^{2}`, `{ab}_{i}`, `10^{-19}`, `\frac{a_{1}+b^{2}}{c}`, `\sqrt{\frac{x^{2}+y^{2}}{2}}`, `\sqrt[3]{x}`, `\bar{U}_{H}`, `\bar{U_{H}}`, `\sqrt{a+√(b)}`, `\_+\^+\{+\}+\\`} {
			for _, source := range []string{"正文" + expression + "结束", `\[` + expression + `\]`, `\(` + expression + `\)`} {
				doc, err := f.Layout(source, s)
				if err != nil {
					t.Fatal(id, source, err)
				}
				joined := ""
				for _, page := range doc.Pages {
					joined += page.Text
					if len(page.Issues) != 0 || len(page.Missing) != 0 {
						t.Fatal(id, source, page.Issues, page.Missing)
					}
				}
				if joined != source {
					t.Fatal("math rewrote stored source", source, joined)
				}
				if _, err := ExportAll(doc, DefaultWriting()); err != nil {
					t.Fatal(id, source, err)
				}
			}
		}
	}
}

func TestMathScriptsShareOneColumnAndRejectDuplicates(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	a, err := f.formulaBox(`x_{12}^{34}`, s)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.formulaBox(`x^{34}_{12}`, s)
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("script order changed geometry", err)
	}
	base, _ := f.formulaBox("x", s)
	sub, _ := f.formulaBox("12", s)
	super, _ := f.formulaBox("34", s)
	if math.Abs(a.width-base.width-max(sub.width, super.width)*.65-s.FontSize*.025) > 1e-8 {
		t.Fatal("scripts are arranged serially")
	}
	for _, source := range []string{`x_{i}_{j}`, `x^{2}^{3}`, `\sqrt{}`, `\sqrt[]{x}`, `\frac{}{2}`, `\frac{2}`, `\bar{U`, `\unknown{a}`, `\sqrt{a+\frac{}{b}}`} {
		doc, err := f.Layout("正文"+source+"结束", s)
		if err != nil {
			t.Fatal("unfinished input prevented editing", source, err)
		}
		if doc.Pages[0].Text != "正文"+source+"结束" || len(doc.Pages[0].Issues) == 0 || len(doc.Pages[0].Diagnostics) == 0 {
			t.Fatal("syntax error was lost", source)
		}
		if _, err := ExportAll(doc, DefaultWriting()); err == nil {
			t.Fatal("invalid math exported", source)
		}
	}
}

func TestCompactInlineFractionsDoNotOverlapRepeatedText(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 8, 30
		s.HandwritingVariation = 100
		text := "前一行文字\n电阻R=\\frac{U_{1}+U_{2}}{I_{g}}，单位Ω。\n后一行文字\n"
		doc, err := f.Layout(strings.Repeat(text, 20), s)
		if err != nil {
			t.Fatal(err)
		}
		joined := ""
		for _, page := range doc.Pages {
			joined += page.Text
			if len(page.Issues) > 0 {
				t.Fatal("inline math collided with prose", id, page.Issues)
			}
		}
		if joined != strings.Repeat(text, 20) {
			t.Fatal("compact math lost source across pages")
		}
		if _, err := ExportAll(doc, DefaultWriting()); err != nil {
			t.Fatal(id, err)
		}
	}
}
