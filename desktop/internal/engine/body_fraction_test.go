package engine

import (
	"math"
	"reflect"
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

func TestBodyFractionFillsRowAndKeepsSharedMathAxis(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		for _, ratio := range []float64{50, 85, 100} {
			for _, variation := range []float64{0, 100} {
				s := DefaultLayout().normalized()
				s.FontID, s.FontSize, s.FractionScale = id, 8.3, ratio
				s.PaperHeight, s.LineCount, s.LineSpacing = 297, 30, 85
				s.MarginTop, s.MarginBottom = 24, 24
				s.LetterSpacing = 70
				s.HandwritingVariation, s.HandwritingSeed = variation, 37
				v := s.variation("body")
				source := `v_{理}=\sqrt{\frac{γRT}{M}}=\sqrt{\frac{γRT_{0}}{M}\frac{T}{T_{0}}}=v_{0}\sqrt{\frac{T}{T_{0}}}`
				p := formulaParser{font: f, input: []rune(source), inline: true, settings: bodyFractionSettings(s), ratio: ratio / 100, variation: v}
				raw, err := p.sequence(0, 0)
				if err != nil {
					t.Fatal(err)
				}
				body, tokens := f.bodyRunes(source, s, v)
				if len(tokens) != 1 || body.bodySource(tokens[0]) != source || body.bodyRuneWidth(tokens[0]) != bodyLength([]rune(source)) {
					t.Fatal("connected equation split or source changed")
				}
				g := body.Glyphs[tokens[0]]
				_, _, rawTop, rawBottom := tableGlyphBounds(Glyph{Strokes: raw.strokes})
				_, _, top, bottom := tableGlyphBounds(g)
				_, pitch := s.rows(0)
				height := (bottom - top) / f.Units * s.FontSize
				if height < .9*pitch || height > pitch || top/f.Units*s.FontSize < bodyInkWidth/2 || bottom/f.Units*s.FontSize > pitch-bodyInkWidth/2 {
					t.Fatalf("equation does not fill row with clearance: %s height=%g pitch=%g", id, height, pitch)
				}
				scale := g.Width / raw.width
				if math.Abs((bottom-top)/(rawBottom-rawTop)-scale) > 1e-8 {
					t.Fatal("equation proportions changed")
				}
				// Every operator, fraction bar and root uses the same transform, so their
				// relative mathematical axes cannot drift when fitting the row.
				shift := top - rawTop*scale
				if len(g.Strokes) != len(raw.strokes) {
					t.Fatal("equation lost strokes")
				}
				for i, stroke := range raw.strokes {
					for j, point := range stroke {
						got := g.Strokes[i][j]
						want := Point{point.X * scale, point.Y*scale + shift}
						if math.Hypot(got.X-want.X, got.Y-want.Y) > 1e-8 {
							t.Fatal("connected equation lost its proportions or shared axis")
						}
					}
				}
				body.fitBodyFractions(s, pitch*1.3)
				body.fitBodyFractions(s, pitch)
				if !reflect.DeepEqual(g, body.Glyphs[tokens[0]]) {
					t.Fatal("refitting accumulated scale errors")
				}
			}
		}
	}
}

func TestBodyFractionsRefitWhenOneParagraphCrossesPages(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize = id, 6
		s.PaperWidth, s.PaperHeight, s.LineCount = 60, 24, 3
		s.HandwritingVariation, s.HandwritingSeed = 100, 37
		s.PageLayouts = []PageLayout{{LineCount: 3}, {LineCount: 4}, {LineCount: 2}}
		source := strings.TrimSpace(strings.Repeat(`x=\frac{a}{b} `, 40))
		doc, err := f.Layout(source, s)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Pages) < 3 {
			t.Fatal("fixture did not cross three page grids")
		}
		joined := ""
		for _, page := range doc.Pages {
			joined += page.Text
			if len(page.Issues) > 0 {
				t.Fatal(id, page.Issues)
			}
			for _, layer := range page.Ink {
				_, _, top, bottom := tableGlyphBounds(Glyph{Strokes: layer.Strokes})
				if bottom-top < .9*page.LineHeight || bottom-top > page.LineHeight {
					t.Fatalf("%s equation retained another page's row height: %g vs %g", id, bottom-top, page.LineHeight)
				}
				row := math.Floor((top + 1e-8) / page.LineHeight)
				if top < row*page.LineHeight+bodyInkWidth/2 || bottom > (row+1)*page.LineHeight-bodyInkWidth/2 {
					t.Fatal("equation crossed a rule")
				}
			}
			for _, cell := range page.Cells {
				if cell.End-cell.Start > 1 {
					before, after := false, false
					for _, caret := range page.Carets {
						before = before || caret.Index == cell.Start
						after = after || caret.Index == cell.End
					}
					if !before || !after {
						t.Fatal("equation lost selection carets")
					}
				}
			}
		}
		if joined != source {
			t.Fatal("page-specific fitting changed source")
		}
		if _, err := ExportAll(doc, DefaultWriting()); err != nil {
			t.Fatal(err)
		}
	}
}
