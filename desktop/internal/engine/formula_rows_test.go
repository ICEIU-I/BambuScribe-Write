package engine

import (
	"math"
	"strings"
	"testing"
)

func TestDisplayFormulasUseOneRuledRowWithoutChangingTheirSource(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		for _, ratio := range []float64{50, 85, 100} {
			for _, variation := range []float64{0, 100} {
				for _, delimiters := range [][2]string{{`\[`, `\]`}, {`\(`, `\)`}} {
					s := DefaultLayout()
					s.FontID, s.FontSize, s.LineCount = id, 8.3, 30
					s.PaperWidth, s.PaperHeight = 210, 297
					s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 24, 24, 17, 17
					s.LetterSpacing, s.LineSpacing, s.FractionScale = 70, 85, ratio
					s.HandwritingVariation, s.HandwritingSeed = variation, 37
					lines := []string{"前一行文字"}
					for _, source := range []string{`F=3πηvd`, `mg=ρ_{0}gV+3πηv_{0}d`, `η=\frac{(ρ−ρ_{0})gd^{2}}{18v_{0}}`, `η=\frac{η}{1+\frac{3}{16}Re}`} {
						lines = append(lines, delimiters[0]+source+delimiters[1])
					}
					lines = append(lines, "后一行文字")
					text := strings.Join(lines, "\n")
					doc, err := f.Layout(text, s)
					if err != nil {
						t.Fatal(err)
					}
					if len(doc.Pages) != 1 || doc.Pages[0].Text != text || len(doc.Pages[0].Issues) != 0 {
						t.Fatal("formula changed source, skipped pages or caused an issue", id)
					}
					page := doc.Pages[0]
					start, row := 0, 0
					for _, cell := range page.Cells {
						for cell.Start >= start+bodyLength([]rune(lines[row])) {
							start += bodyLength([]rune(lines[row])) + 1
							row++
						}
						want := s.MarginTop + float64(row)*page.LineHeight
						tolerance := 1e-8
						if row == 0 || row == len(lines)-1 {
							tolerance = s.FontSize * .1 // Ordinary handwriting retains its baseline variation.
						}
						if math.Abs(cell.Y-want) > tolerance || cell.Height > page.LineHeight+1e-8 {
							t.Fatalf("%s row %d occupies extra space: %+v, expected Y %g", id, row, cell, want)
						}
					}
					for i, formula := range page.Formulas {
						y := s.MarginTop + float64(i+1)*page.LineHeight
						if formula.Y < y+bodyInkWidth/2 || formula.Y+formula.Height > y+page.LineHeight-bodyInkWidth/2 {
							t.Fatal("formula ink touches the neighbouring row", id, formula)
						}
					}
					if _, err := ExportAll(doc, DefaultWriting()); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}

func TestDisplayEquationRefitsAtTheNextPageGridAndKeepsProportions(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize = id, 6
		s.PaperWidth, s.PaperHeight, s.LineCount = 110, 40, 4
		s.PageLayouts = []PageLayout{{LineCount: 4}, {LineCount: 8}}
		source := `η=\frac{(ρ−ρ_{0})gd^{2}}{18v_{0}(1+\frac{2.4d}{D})}`
		text := "正文\n正文\n正文\n正文\n\\[" + source + `\]` + "\n结束"
		doc, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Pages) != 2 || len(doc.Pages[1].Formulas) != 1 {
			t.Fatal("equation did not fit into the first row of the next page", id)
		}
		page := doc.Pages[1]
		if doc.Pages[0].Text+page.Text != text || page.Cells[0].Y != 0 || page.Cells[1].Y != page.LineHeight {
			t.Fatal("page transition changed source or row positions")
		}
		raw, err := f.formulaBox(source, s)
		if err != nil {
			t.Fatal(err)
		}
		layer := page.Ink[0]
		formula := page.Formulas[0]
		scale := formula.Width / raw.width
		if math.Abs(formula.Height/(raw.bottom-raw.top)-scale) > 1e-8 {
			t.Fatal("formula was stretched on one axis")
		}
		for i, stroke := range raw.strokes {
			for j, point := range stroke {
				want := Point{formula.X + point.X*scale, formula.Y + (point.Y-raw.top)*scale}
				got := layer.Strokes[i][j]
				if math.Hypot(want.X-got.X, want.Y-got.Y) > 1e-8 {
					t.Fatal("formula lost proportions or shared mathematical axes")
				}
			}
		}
		code, err := ExportPage(doc, 1, DefaultWriting())
		if err != nil {
			t.Fatal(err)
		}
		written := replayWrittenStrokes(t, string(code), Point{Y: s.PaperHeight}, *DefaultWriting().PenDown)
		for i, stroke := range page.Strokes {
			for j, point := range stroke {
				want := Point{point.X, s.PaperHeight - point.Y}
				if math.Hypot(want.X-written[i][j].X, want.Y-written[i][j].Y) > .00072 {
					t.Fatal("export differs from preview")
				}
			}
		}
	}
}
