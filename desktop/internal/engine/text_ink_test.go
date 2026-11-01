package engine

import (
	"os"
	"reflect"
	"testing"
)

func compactDCLayout() LayoutSettings {
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight = 210, 297
	s.MarginTop, s.MarginBottom, s.MarginLeft, s.MarginRight = 25.4, 25.4, 17, 17
	s.LineCount = 30
	s.FontSize = (297.0 - 2*25.4) / 30
	s.LetterSpacing, s.LineSpacing, s.FontID = 76.5, 85, FontDC
	return s
}

func TestCompactDCChecksHorizontalInkPosition(t *testing.T) {
	f, _ := BuiltinFont(FontDC)
	s := compactDCLayout()
	for _, align := range []string{"left", "center", "right"} {
		s.Align = align
		d, err := f.Layout("料       \n       着", s)
		if err != nil {
			t.Fatalf("disjoint descender/ascender blocked %s alignment: %v", align, err)
		}
		if len(d.Pages) != 1 || len(d.Missing) != 0 || d.Settings.LineSpacing != 85 || d.Settings.FontSize != s.FontSize {
			t.Fatal("fix changed the layout settings")
		}
		// The first character must retain every source point and proportion.
		for i, stroke := range f.Glyphs['料'].Strokes {
			got := d.Pages[0].Strokes[i]
			for j := range stroke {
				if !closeFloat(got[j].X-got[0].X, (stroke[j].X-stroke[0].X)*s.FontSize/f.Units) ||
					!closeFloat(got[j].Y-got[0].Y, (stroke[j].Y-stroke[0].Y)*s.FontSize/f.Units) {
					t.Fatal("overlap fix scaled or deformed the font")
				}
			}
		}
	}
	requireOverlapIssue(t, f, "料\n着", s)
}

func TestCompactDCLessonKeepsOriginalSettingsAndExports(t *testing.T) {
	text, err := os.ReadFile("testdata/dc-compact-lesson.txt")
	if err != nil {
		t.Fatal(err)
	}
	s := compactDCLayout()
	s.PageLayouts = make([]PageLayout, 5)
	for i := range s.PageLayouts {
		s.PageLayouts[i].LineCount = 30
	}
	d, err := testFont(t).Layout(string(text), s)
	if err != nil || len(d.Missing) != 0 || !reflect.DeepEqual(d.Settings, s.normalized()) {
		t.Fatal("actual compact lesson cannot use DC with its original parameters", err)
	}
	if _, err := ExportAll(d, DefaultWriting()); err != nil {
		t.Fatal(err)
	}
}

func TestInkTrackerRetainsLongDescendersAndResetsPerPage(t *testing.T) {
	f := &Font{Units: 11, Glyphs: map[rune]Glyph{
		'a': {Width: 2, Strokes: []Stroke{{{0, 9}, {2, 11}}}},
		'b': {Width: 2, Strokes: []Stroke{{{2, 0}, {0, 2}}}},
		'c': {Width: 2, Strokes: []Stroke{{{0, 5}, {2, 6}}}},
	}}
	s := DefaultLayout()
	s.LineSpacing = 50
	if _, err := f.Layout("a\n       c", s); err != nil {
		t.Fatal("intermediate disjoint line should fit", err)
	}
	for _, text := range []string{"a\n       c\nb", "a\n\nb"} {
		requireOverlapIssue(t, f, text, s)
	}
	if _, err := f.Layout("a\fb", s); err != nil {
		t.Fatal("ink from another sheet blocked its first row", err)
	}
}

func closeFloat(a, b float64) bool { return a-b < 1e-8 && b-a < 1e-8 }
