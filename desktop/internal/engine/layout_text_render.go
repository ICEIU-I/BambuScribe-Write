package engine

import (
	"slices"
	"strconv"
	"strings"
	"unicode"
)

func (f *Font) appendTextFragment(page *Page, runes []rune, part textFragment, s LayoutSettings, y float64, textIndex *int, missingSet map[rune]bool, doc *Document, variation handwritingVariation) {
	x := textFragmentOrigin(part, s)
	var text strings.Builder
	page.Carets = append(page.Carets, CaretStop{*textIndex, x, y})
	for _, r := range runes {
		source, length := f.bodySource(r), f.bodyRuneWidth(r)
		advance := f.advance(r, s)
		page.Cells = append(page.Cells, GlyphCell{*textIndex, *textIndex + length, x, y, min(advance, part.right-x), min(s.FontSize, page.LineHeight, s.PaperHeight-s.MarginBottom-y)})
		if f.bodyScripts[r].math {
			_, _, top, bottom := tableGlyphBounds(f.Glyphs[r])
			cell := &page.Cells[len(page.Cells)-1]
			top = min(0, top/f.Units*s.FontSize)
			cell.Y = y + top
			cell.Height = max(cell.Height, bottom/f.Units*s.FontSize) - top
		}
		if !unicode.IsSpace(r) {
			g, ok := f.Glyphs[r]
			if !ok {
				page.Missing = append(page.Missing, MissingGlyph{string(r), x, y, s.FontSize})
				if !missingSet[r] {
					missingSet[r] = true
					doc.Missing = append(doc.Missing, string(r))
				}
			} else {
				bearing := textGlyphBearing(g, f.Units, advance, s)
				bearing += f.letterOffset(g, s.FontSize, bearing, advance, variation, *textIndex)
				start := len(page.Strokes)
				for _, stroke := range f.bodyStrokes(r, g, variation, *textIndex) {
					out := make(Stroke, len(stroke))
					for i, p := range stroke {
						out[i] = Point{x + bearing + p.X/f.Units*s.FontSize, y + p.Y/f.Units*s.FontSize}
					}
					page.Strokes = append(page.Strokes, out)
				}
				page.Ink = append(page.Ink, InkLayer{"", strconv.Itoa(*textIndex), inkSource([]string{f.ID, source}), page.Strokes[start:]})
			}
		}
		for _, issue := range f.bodyIssues {
			if *textIndex >= f.bodyIssueOffset+issue.start && *textIndex < f.bodyIssueOffset+issue.end {
				if !slices.Contains(page.Issues, issue.message) {
					page.Issues = append(page.Issues, issue.message)
				}
				page.Diagnostics = append(page.Diagnostics, LayoutDiagnostic{Code: "math-syntax", Message: issue.message, RegionBounds: RegionBounds{X: x, Y: y, Width: advance, Height: s.FontSize}})
			}
		}
		x += advance
		*textIndex += length
		page.Carets = append(page.Carets, CaretStop{*textIndex, min(x, part.right), y})
		text.WriteString(source)
	}
	page.Text += text.String()
}
