package engine

import (
	"strings"
	"testing"
)

func TestDelimitedMathCanJoinProseWithoutLosingSourceOrSelection(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 8.3, 30
		s.PaperWidth, s.PaperHeight = 210, 297
		s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 24, 24, 17, 17
		s.LineSpacing, s.LetterSpacing = 85, 70
		for _, pair := range [][2]string{{`\[`, `\]`}, {`\(`, `\)`}} {
			for _, formula := range []string{`Re=\frac{v_{0}dρ_{0}}{η}`, `x=\sqrt{\frac{x_{i}^{2}+y^{2}}{2}}`, `U=\frac{1}{2}+\frac{3}{4}`} {
				prefix, suffix := "前文：", "，继续正文。"
				span := pair[0] + formula + pair[1]
				text := prefix + span + suffix + "\n后一行"
				doc, err := f.Layout(text, s)
				if err != nil {
					t.Fatal(err)
				}
				if len(doc.Pages) != 1 || doc.Pages[0].Text != text || len(doc.Pages[0].Issues) != 0 || len(doc.Missing) != 0 {
					t.Fatal("delimited inline math caused errors or changed the source", id, text)
				}
				page := doc.Pages[0]
				found := false
				for _, cell := range page.Cells {
					if cell.Start == bodyLength([]rune(prefix)) && cell.End == bodyLength([]rune(prefix+span)) {
						found = true
						if cell.Height > page.LineHeight+1e-8 {
							t.Fatal("delimited fraction occupied an extra row")
						}
					}
				}
				if !found {
					t.Fatal("selection split or omitted formula delimiters")
				}
				if _, err := ExportAll(doc, DefaultWriting()); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestInlineDelimiterErrorsKeepSourceAndGiveSpecificGuidance(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 5
	for _, span := range []string{`\[x=1`, `\(x=1`, `\[x=1\)`, `\(x=1\]`, `\[\]`} {
		text := "正文：" + span
		doc, err := f.Layout(text, s)
		if err != nil || doc.Pages[0].Text != text || len(doc.Pages[0].Issues) == 0 {
			t.Fatal("unfinished math prevented editing or lost its source", text, err)
		}
		if strings.Contains(strings.Join(doc.Pages[0].Issues, ""), "不支持命令") {
			t.Fatal("delimiter error reported as an unknown command")
		}
		if _, err := ExportAll(doc, DefaultWriting()); err == nil {
			t.Fatal("unfinished math was exported")
		}
	}
}
