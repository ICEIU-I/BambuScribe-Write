package engine

import (
	"math"
	"strings"
	"testing"
)

func TestBodyFractionsStayOnConsecutiveWritingRows(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 8.3, 30
		s.PaperWidth, s.PaperHeight = 210, 297
		s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 24, 24, 17, 17
		s.LetterSpacing, s.LineSpacing = 70, 85
		lines := []string{"前一行文字"}
		for _, formula := range []string{`\frac{2Πx}{λ}`, `\frac{U_{1}+U_{2}}{I_{g}}`, `\sqrt{\frac{x^{2}+y^{2}}{2}}`, `\frac{\frac{a}{b}+c}{d}^{2}`} {
			lines = append(lines, "正文"+formula+"结束")
		}
		lines = append(lines, "后一行文字")
		text := strings.Join(lines, "\n")
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Pages) != 1 || doc.Pages[0].Text != text || len(doc.Pages[0].Issues) > 0 {
			t.Fatal("fraction changed source or caused overflow", id, doc.Pages)
		}
		page := doc.Pages[0]
		row, start := 0, 0
		for _, cell := range page.Cells {
			for cell.Start >= start+bodyLength([]rune(lines[row])) {
				start += bodyLength([]rune(lines[row])) + 1
				row++
			}
			wantY := s.MarginTop + float64(row)*page.LineHeight
			if math.Abs(cell.Y-wantY) > 1e-8 {
				t.Fatalf("%s row %d skipped a ruled row: y=%g want=%g", id, row, cell.Y, wantY)
			}
			if cell.Height > page.LineHeight+1e-8 {
				t.Fatalf("%s selection extends into the next row", id)
			}
		}
		w := DefaultWriting()
		code, err := ExportPage(doc, 0, w)
		if err != nil {
			t.Fatal(err)
		}
		written := replayWrittenStrokes(t, string(code), Point{Y: s.PaperHeight}, *w.PenDown)
		if len(written) != len(page.Strokes) {
			t.Fatal("export lost fraction strokes")
		}
		for i, stroke := range page.Strokes {
			if len(written[i]) != len(stroke) {
				t.Fatal("export lost fraction points")
			}
			for j, p := range stroke {
				want := Point{p.X - s.MarginLeft, s.PaperHeight - p.Y + s.MarginTop}
				if math.Hypot(written[i][j].X-want.X, written[i][j].Y-want.Y) > .00072 {
					t.Fatal("export changed the compact fraction geometry")
				}
			}
		}
	}
}

func TestBodyFractionKeepsProportionsAndItsMathAxis(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		for _, ratio := range []float64{50, 85, 100} {
			for _, variation := range []float64{0, 100} {
				s := DefaultLayout().normalized()
				s.FontID, s.FontSize, s.FractionScale = id, 8.3, ratio
				s.HandwritingVariation, s.HandwritingSeed = variation, 37
				v := s.variation("body")
				source := `\frac{2Πx}{λ}`
				p := formulaParser{font: f, input: []rune(source), inline: true, settings: s, ratio: ratio / 100, variation: v}
				raw, err := p.atom(0)
				if err != nil {
					t.Fatal(err)
				}
				body, tokens := f.bodyRunes(source, s, v)
				if len(tokens) != 1 {
					t.Fatal("fraction was split")
				}
				g := body.Glyphs[tokens[0]]
				scale := g.Width / raw.width
				if scale >= 1 {
					t.Fatal("tall fraction was not made compact")
				}
				if len(g.Strokes) != len(raw.strokes) {
					t.Fatal("fraction lost strokes")
				}
				for i, stroke := range raw.strokes {
					for j, point := range stroke {
						got := g.Strokes[i][j]
						want := Point{point.X * scale, f.Units + p.axis() + (point.Y-p.axis())*scale}
						if math.Hypot(got.X-want.X, got.Y-want.Y) > 1e-8 {
							t.Fatal("fraction was stretched or moved off its math axis")
						}
						if got.Y < 0 || got.Y > f.Units {
							t.Fatal("fraction still exceeds the body cell", id, got)
						}
					}
				}
				display, err := f.formulaBox(source, s)
				if err != nil {
					t.Fatal(err)
				}
				_, _, top, bottom := tableGlyphBounds(g)
				if display.bottom-display.top <= (bottom-top)/f.Units*s.FontSize {
					t.Fatal("display formula was also compressed")
				}
			}
		}
	}
}
