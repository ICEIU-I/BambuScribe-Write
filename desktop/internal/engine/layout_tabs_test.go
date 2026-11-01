package engine

import (
	"math"
	"testing"
)

func TestTabAdvancesFourWritingCellsAndKeepsExportGeometry(t *testing.T) {
	for _, font := range []string{FontHandwriting, FontDC} {
		f, err := BuiltinFont(font)
		if err != nil {
			t.Fatal(err)
		}
		for _, percent := range []float64{72.5, 100, 150} {
			s := DefaultLayout()
			s.FontSize, s.LetterSpacing, s.LetterGap = 6, percent, .2
			s.FontID = font
			doc, err := f.Layout("你\t四\n\t正文", s)
			if err != nil {
				t.Fatal(err)
			}
			p := doc.Pages[0]
			want := 4 * (s.FontSize*percent/100 + s.LetterGap)
			for _, index := range []int{1, 3} {
				a, b := p.Cells[index], p.Cells[index+1]
				if math.Abs(b.X-a.X-want) > 1e-8 || a.Y != b.Y {
					t.Fatalf("tab at %s/%.1f%% moved %.4f instead of four cells %.4f", font, percent, b.X-a.X, want)
				}
			}
			if p.Text != "你\t四\n\t正文" || len(p.Missing) != 0 {
				t.Fatal("tab changed the editable text or became a missing glyph")
			}
			if _, err := ExportAll(doc, testWriting()); err != nil {
				t.Fatal("tabbed text could not export", err)
			}
		}
	}
}

func TestTabNearLineEndWrapsWithItsFourCellWidth(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.PaperWidth, s.FontSize = 40, 6
	doc, err := f.Layout("你好你好\t四", s)
	if err != nil {
		t.Fatal(err)
	}
	cells := doc.Pages[0].Cells
	if cells[4].Y <= cells[3].Y || cells[4].X != s.MarginLeft || math.Abs(cells[5].X-cells[4].X-24) > 1e-8 {
		t.Fatal("line-end tab was lost or advanced by the wrong width", cells)
	}
}
