package engine

import (
	"strconv"
	"unicode"
)

func (f *Font) appendTextFragment(page *Page, runes []rune, part textFragment, s LayoutSettings, y float64, textIndex *int, missingSet map[rune]bool, doc *Document) {
	x := textFragmentOrigin(part, s)
	page.Carets = append(page.Carets, CaretStop{*textIndex, x, y})
	for _, r := range runes {
		advance := f.advance(r, s)
		page.Cells = append(page.Cells, GlyphCell{*textIndex, *textIndex + utf16Width(r), x, y, min(advance, part.right-x), min(s.FontSize, page.LineHeight, s.PaperHeight-s.MarginBottom-y)})
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
				start := len(page.Strokes)
				for _, stroke := range g.Strokes {
					out := make(Stroke, len(stroke))
					for i, p := range stroke {
						out[i] = Point{x + bearing + p.X/f.Units*s.FontSize, y + p.Y/f.Units*s.FontSize}
					}
					page.Strokes = append(page.Strokes, out)
				}
				page.Ink = append(page.Ink, InkLayer{"", strconv.Itoa(*textIndex), inkSource([]string{f.ID, string(r)}), page.Strokes[start:]})
			}
		}
		x += advance
		*textIndex += utf16Width(r)
		page.Carets = append(page.Carets, CaretStop{*textIndex, min(x, part.right), y})
	}
	page.Text += string(runes)
}
