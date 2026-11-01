package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestBodySubscriptKeepsRuledRowsAndNormalGlyphs(t *testing.T) {
	for _, id := range []string{FontHandwriting, FontDC} {
		f, _ := BuiltinFont(id)
		s := DefaultLayout()
		s.FontID, s.FontSize, s.LineCount = id, 8.3, 30
		s.PaperWidth, s.PaperHeight = 210, 297
		s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 24, 24, 17, 17
		s.LetterSpacing, s.LineSpacing = 70, 85
		text := "内阻R_{g}.\n在表头上并联分流电阻R_{s}.\nI_{g} R_{g} = (I − I_{g})R_{s}\n正文继续\nΔI = I_{改} − I_{标}"
		doc, err := f.Layout(text, s)
		if err != nil || len(doc.Pages) != 1 {
			t.Fatal(id, err)
		}
		page := doc.Pages[0]
		if page.Text != text || len(page.Formulas) != 0 || len(page.Issues) != 0 || len(doc.Missing) != 0 {
			t.Fatal("inline scripts changed source or became display formulas", id, page.Issues)
		}
		for _, cell := range page.Cells {
			row := strings.Count(string([]rune(text)[:cell.Start]), "\n")
			want := s.MarginTop + float64(row)*page.LineHeight
			if math.Abs(cell.Y-want) > 1e-8 {
				t.Fatalf("%s row drift: got %g want %g", id, cell.Y, want)
			}
		}
		plain, _ := f.Layout("内阻R\n后续", s)
		// The normal glyph preceding a subscript keeps the ordinary body baseline.
		normal := len(f.Glyphs['内'].Strokes) + len(f.Glyphs['阻'].Strokes) + len(f.Glyphs['R'].Strokes)
		for i, stroke := range page.Strokes[:normal] {
			for j, point := range stroke {
				want := plain.Pages[0].Strokes[i][j]
				if math.Abs(point.X-want.X) > 1e-8 || math.Abs(point.Y-want.Y) > 1e-8 {
					t.Fatal("adding a subscript moved or resized normal glyphs", id, point, want)
				}
			}
		}
		if _, err := ExportAll(doc, DefaultWriting()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBodySubscriptPreservesSourceIndicesAcrossWrappingAndPages(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize, s.PaperWidth, s.PaperHeight, s.LineCount = 5, 55, 45, 5
	text := strings.Repeat("正文I_{改}和U_{标} ", 16) + "\fR_{gm}=I_{g}"
	doc, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, page := range doc.Pages {
		joined += page.Text
		for _, cell := range page.Cells {
			span := string([]rune(text)[cell.Start:cell.End])
			if strings.Contains(span, "_") && !strings.HasSuffix(span, "}") {
				t.Fatal("script split across cells", span)
			}
			if cell.X+cell.Width > s.PaperWidth+1e-8 {
				t.Fatal("script outside page")
			}
		}
	}
	if joined != text || len(doc.Pages) < 2 {
		t.Fatal("source lost during pagination")
	}
	// Supplementary source characters still count as two UTF-16 code units.
	font := *f
	font.Glyphs = map[rune]Glyph{'𠀀': f.Glyphs['中'], 'g': f.Glyphs['g']}
	unicodeDoc, err := font.Layout("𠀀_{g}𠀀", s)
	if err != nil {
		t.Fatal(err)
	}
	cells := unicodeDoc.Pages[0].Cells
	if len(cells) != 2 || cells[0].End != 6 || cells[1].Start != 6 || cells[1].End != 8 {
		t.Fatal("source caret offsets are not UTF-16", cells)
	}
}

func TestBodySubscriptFallbackAndFontIsolation(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 5
	before, _ := f.Layout("正文R_g", s)
	for _, source := range []string{"R_{}", "R_{g", "R_{ }", "R_{x_{g}}"} {
		font, runes := f.bodyRunes(source, s)
		if font != f || string(runes) != source {
			t.Fatal("invalid group was interpreted", source)
		}
	}
	font, _ := f.bodyRunes("R_{g} I_{改} U_{标}", s)
	if font == f || len(f.bodyScripts) != 0 {
		t.Fatal("shared font was modified")
	}
	after, _ := f.Layout("正文R_g", s)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("plain body layout changed")
	}
	doc, err := f.Layout("R_{🧪}", s)
	if err != nil || len(doc.Missing) == 0 {
		t.Fatal("unsupported subscript silently disappeared", err)
	}
	if _, err := ExportAll(doc, DefaultWriting()); err == nil {
		t.Fatal("missing glyph exported")
	}
}
