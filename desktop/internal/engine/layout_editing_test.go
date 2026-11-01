package engine

import (
	"strings"
	"testing"
)

func requireOverlapIssue(t *testing.T, f *Font, text string, s LayoutSettings) *Document {
	t.Helper()
	doc, err := f.Layout(text, s)
	if err != nil {
		t.Fatalf("overlapping ink must keep the editable preview: %v", err)
	}
	found := false
	for _, page := range doc.Pages {
		for _, issue := range page.Issues {
			if strings.Contains(issue, "重叠") {
				found = true
				if _, err := GenerateGCode(page, doc.Settings, DefaultWriting()); err == nil || !strings.Contains(err.Error(), "重叠") {
					t.Fatal("overlapping ink must still block output", err)
				}
			}
		}
	}
	if !found {
		t.Fatal("overlapping ink lost its page issue")
	}
	return doc
}

func TestEnterKeepsNewTextAndCaretWhenLaterLinesOverlap(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.LineSpacing = 50
	text := "你\n你\n四、实验内容及步骤"
	doc := requireOverlapIssue(t, f, text, s)
	if got := doc.Pages[0].Text; got != text {
		t.Fatalf("preview kept stale text: %q", got)
	}
	var heading *GlyphCell
	// Two Chinese characters and two newlines precede the heading.
	for i := range doc.Pages[0].Cells {
		if doc.Pages[0].Cells[i].Start == 4 {
			heading = &doc.Pages[0].Cells[i]
		}
	}
	if heading == nil || heading.Y <= doc.Pages[0].Cells[1].Y {
		t.Fatal("heading did not move onto its new line")
	}
	found := false
	for _, caret := range doc.Pages[0].Carets {
		if caret.Index == 4 && caret.X == heading.X && caret.Y == heading.Y {
			found = true
		}
	}
	if !found {
		t.Fatal("caret before 四 was lost")
	}
	s.LineSpacing = 100
	fixed, err := f.Layout(text, s)
	if err != nil || len(fixed.Pages[0].Issues) != 0 {
		t.Fatal("corrected spacing kept the previous issue", err)
	}
	if _, err := ExportAll(fixed, DefaultWriting()); err != nil {
		t.Fatal("corrected document could not export", err)
	}
}
