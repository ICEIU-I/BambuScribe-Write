package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestTrailingBlankCaretsMatchActualInsertion(t *testing.T) {
	f, _ := EmbeddedFont()
	s := DefaultLayout()
	s.FontSize, s.LineCount, s.Indent, s.ParagraphGap = 5, 12, 2, 1.5
	for _, text := range []string{
		"正文", "正文\n", "正文\n\n", "前页\f正文\f后页", "前页\f正文\n\f后页",
		"前页𠀀\f" + strings.Repeat("正文", 30) + "\n结尾\f后页", "\f\f后页",
	} {
		t.Run(text[:min(len(text), 30)], func(t *testing.T) {
			doc, err := f.Layout(text, s)
			if err != nil {
				t.Fatal(err)
			}
			checked := 0
			for pi, page := range doc.Pages {
				for _, stop := range page.BlankCarets {
					units := utf16.Encode([]rune(text))
					before, after := string(utf16.Decode(units[:stop.Index])), string(utf16.Decode(units[stop.Index:]))
					inserted := strings.Repeat("\n", stop.BlankLines) + "字"
					next, err := f.Layout(before+inserted+after, s)
					if err != nil {
						t.Fatal(err)
					}
					if len(next.Pages) != len(doc.Pages) {
						t.Fatalf("blank insertion shifted pages: %d -> %d", len(doc.Pages), len(next.Pages))
					}
					found := false
					for _, cell := range next.Pages[pi].Cells {
						if cell.Start == stop.Index+stop.BlankLines {
							found = true
							if math.Abs(cell.X-stop.X) > 1e-8 || math.Abs(cell.Y-stop.Y) > 1e-8 {
								t.Fatalf("caret %+v does not match inserted cell %+v", stop, cell)
							}
						}
					}
					if !found {
						t.Fatalf("no inserted cell on page %d at %+v", pi, stop)
					}
					for other := range doc.Pages {
						if other != pi && !reflect.DeepEqual(next.Pages[other].Strokes, doc.Pages[other].Strokes) {
							t.Fatal("unrelated page changed")
						}
					}
					checked++
				}
			}
			if checked == 0 {
				t.Fatal("no selectable blank rows")
			}
		})
	}
}

func TestTrailingBlankCaretsAvoidObjectsAndRespectRowLimit(t *testing.T) {
	f, _ := EmbeddedFont()
	s := DefaultLayout()
	s.FontSize, s.LineCount, s.Indent = 5, 4, 0
	s.LineSpacing = 100
	pitch := (s.PaperHeight - s.MarginTop - s.MarginBottom) / 4
	occupied := map[int][]RegionBounds{0: {{s.MarginLeft, s.MarginTop + pitch, s.PaperWidth - s.MarginLeft - s.MarginRight, pitch / 2}}}
	doc, err := f.layoutText("正文\f后页", s, occupied)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].BlankCarets) != 2 {
		t.Fatalf("expected two unblocked rows: %+v", doc.Pages[0].BlankCarets)
	}
	for _, stop := range doc.Pages[0].BlankCarets {
		if stop.BlankLines < 2 {
			t.Fatal("blocked row ignored in newline count")
		}
		next, err := f.layoutText("正文"+strings.Repeat("\n", stop.BlankLines)+"字\f后页", s, occupied)
		if err != nil {
			t.Fatal(err)
		}
		last := next.Pages[0].Cells[len(next.Pages[0].Cells)-1]
		if math.Abs(last.Y-stop.Y) > 1e-8 {
			t.Fatal("object avoidance moved insertion")
		}
	}
	full, err := f.Layout(strings.Repeat("正文\n", 4)+"\f后页", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Pages[0].BlankCarets) != 0 {
		t.Fatal("full page offered a blank row")
	}
}
