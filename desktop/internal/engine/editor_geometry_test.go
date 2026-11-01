package engine

import "testing"

func TestCaretOffsetsRespectUTF16AndNewlines(t *testing.T) {
	f := testFont(t)
	d, err := f.Layout("你🙂\n\n好", DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	page := d.Pages[0]
	if len(page.Cells) != 3 {
		t.Fatal("wrong cell count")
	}
	if page.Cells[1].Start != 1 || page.Cells[1].End != 3 || page.Cells[2].Start != 5 || page.Cells[2].End != 6 {
		t.Fatal("selection offsets differ from browser UTF-16")
	}
	seen := map[int]bool{}
	for _, c := range page.Carets {
		seen[c.Index] = true
	}
	for _, index := range []int{0, 1, 3, 4, 5, 6} {
		if !seen[index] {
			t.Errorf("missing caret %d", index)
		}
	}
}

func TestCaretForEmptyDocumentAndAlignment(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	d, _ := f.Layout("", s)
	if len(d.Pages[0].Carets) != 1 || d.Pages[0].Carets[0].Index != 0 {
		t.Fatal("empty page not editable")
	}
	left, _ := f.Layout("你好", s)
	s.Align = "center"
	center, _ := f.Layout("你好", s)
	s.Align = "right"
	right, _ := f.Layout("你好", s)
	if !(left.Pages[0].Cells[0].X < center.Pages[0].Cells[0].X && center.Pages[0].Cells[0].X < right.Pages[0].Cells[0].X) {
		t.Fatal("alignment not reflected in geometry")
	}
	if right.Pages[0].Carets[2].X > s.PaperWidth-s.MarginRight+1e-8 {
		t.Fatal("aligned caret outside page")
	}
}
