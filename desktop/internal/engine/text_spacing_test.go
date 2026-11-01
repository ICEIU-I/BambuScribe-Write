package engine

import (
	"math"
	"strings"
	"testing"
)

func TestSpacingChangesPitchWithoutChangingGlyphGeometry(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight = 210, 297
	s.MarginLeft, s.MarginRight, s.MarginTop, s.MarginBottom = 25.4, 25.4, 25.4, 25.4
	s.LineCount = 30
	s.FontSize = (297.0 - 25.4 - 25.4) / 30
	before, err := f.Layout("你好\n你好", s)
	if err != nil {
		t.Fatal(err)
	}
	s.LetterSpacing, s.LineSpacing = 85, 85
	after, err := f.Layout("你好\n你好", s)
	if err != nil {
		t.Fatal(err)
	}
	a, b := before.Pages[0], after.Pages[0]
	if math.Abs((b.Cells[1].X-b.Cells[0].X)/(a.Cells[1].X-a.Cells[0].X)-.85) > 1e-9 || math.Abs((b.Cells[2].Y-b.Cells[0].Y)/(a.Cells[2].Y-a.Cells[0].Y)-.85) > 1e-9 {
		t.Fatal("character and row pitch were not reduced by 15%")
	}
	if len(a.Strokes) != len(b.Strokes) || after.Settings.FontSize != before.Settings.FontSize {
		t.Fatal("spacing changed font size or lost strokes")
	}
	for i, stroke := range a.Strokes {
		other := b.Strokes[i]
		if len(stroke) != len(other) {
			t.Fatal("spacing changed glyph points")
		}
		for j, point := range stroke {
			if math.Abs((point.X-stroke[0].X)-(other[j].X-other[0].X)) > 1e-9 || math.Abs((point.Y-stroke[0].Y)-(other[j].Y-other[0].Y)) > 1e-9 {
				t.Fatal("spacing scaled or deformed ink")
			}
		}
	}
	writing := testWriting()
	writing.OriginMode = "auto"
	code, err := ExportPage(after, 0, writing)
	if err != nil {
		t.Fatal(err)
	}
	moves := xyMoves(t, string(code), Point{})
	index := 0
	for _, stroke := range b.Strokes {
		for _, p := range stroke {
			want := Point{p.X - s.MarginLeft, -(p.Y - s.MarginTop)}
			if math.Abs(moves[index].X-want.X) > .000501 || math.Abs(moves[index].Y-want.Y) > .000501 {
				t.Fatal("uploaded spacing differs from preview")
			}
			index++
		}
	}
}

func Test110PercentTypeFitsItsActualInkAndAllConfiguredRows(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	doc, err := f.Layout(strings.TrimSuffix(strings.Repeat("你好\n", 16), "\n"), s)
	if err != nil {
		t.Fatal(err)
	}
	if s.FontSize/s.LineHeight != 1.1 || len(doc.Pages) != 1 || doc.Pages[0].LineCount != 16 || len(doc.Pages[0].Cells) != 32 {
		t.Fatal("110% type lost rows or was blocked by its em height")
	}
	for _, cell := range doc.Pages[0].Cells {
		if cell.Height > doc.Pages[0].LineHeight || cell.Y+cell.Height > s.PaperHeight-s.MarginBottom+1e-8 {
			t.Fatal("selection boxes overlap rows or leave the writing area")
		}
	}
	for _, stroke := range doc.Pages[0].Strokes {
		for _, p := range stroke {
			if p.Y > s.PaperHeight-s.MarginBottom+1e-8 {
				t.Fatal("actual ink extends below the paper")
			}
		}
	}
	// A 16.5 mm em is also valid in a 15 mm writing area when the ink fits.
	s.PaperHeight, s.MarginTop, s.MarginBottom, s.LineCount = 20, 2.5, 2.5, 1
	if _, err := f.Layout("你好", s); err != nil {
		t.Fatal(err)
	}
}

func TestA4RowRoundingAndCompactPagination(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.PaperHeight, s.MarginTop, s.MarginBottom, s.LineCount = 297, 25.4, 25.4, 30
	s.FontSize = (297.0 - 25.4 - 25.4) / 30
	for _, percent := range []float64{100, 85} {
		s.LineSpacing = percent
		want := 30
		if percent == 85 {
			want = 35
		}
		doc, err := f.Layout(strings.TrimSuffix(strings.Repeat("你好\n", want+1), "\n"), s)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Pages) != 2 || doc.Pages[0].LineCount != want || len(doc.Pages[0].Cells) != want*2 || len(doc.Pages[1].Cells) != 2 {
			t.Fatalf("incorrect page grid at %.0f%%: %+v", percent, doc.Pages[0])
		}
	}
}

func TestRandomSingleRowKeepsFinitePitch(t *testing.T) {
	s := DefaultLayout()
	s.LineMode, s.LineCount, s.LineSpacing = "random", 1, 85
	for page := range 100 {
		count, pitch := s.rows(page)
		if count < 1 || !finite(pitch) || pitch <= 0 {
			t.Fatalf("random single-row page %d has invalid grid: %d / %v", page, count, pitch)
		}
	}
}

func TestSpacingRejectsRealOverlapAndKeepsLegacyGrid(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.LineSpacing = 50
	requireOverlapIssue(t, f, "你\n你", s)
	s = DefaultLayout()
	s.LetterSpacing = 50
	doc, err := f.Layout("你你", s)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Cells[0].Width < f.Glyphs['你'].Width/f.Units*s.FontSize {
		t.Fatal("tight character pitch overlaps wide glyphs")
	}
	s = DefaultLayout()
	s.LetterSpacing, s.LineSpacing = 0, 0
	legacy, err := f.Layout("你好", s)
	if err != nil || legacy.Settings.LetterSpacing != 100 || legacy.Settings.LineSpacing != 100 {
		t.Fatalf("legacy projects must keep 100%% spacing: %v", err)
	}
	for _, invalid := range []float64{-1, 49, 201, math.NaN()} {
		s = DefaultLayout()
		s.LineSpacing = invalid
		if _, err := f.Layout("你好", s); err == nil {
			t.Fatalf("invalid line spacing accepted: %v", invalid)
		}
	}
}
