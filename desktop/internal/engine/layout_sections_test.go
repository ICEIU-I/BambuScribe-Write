package engine

import (
	"reflect"
	"strings"
	"testing"
)

func TestExplicitPagesPreserveGridAndSourceOffsets(t *testing.T) {
	font, err := EmbeddedFont()
	if err != nil {
		t.Fatal(err)
	}
	s := DefaultLayout()
	s.FontSize, s.LineCount, s.Indent = 5, 4, 2
	s.PageLayouts = []PageLayout{{LineCount: 4}, {LineCount: 4}}
	one, err := font.Layout("短页正文", s)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := font.Layout("短页正文\f短页正文", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Pages) != 2 {
		t.Fatalf("got %d pages", len(copy.Pages))
	}
	for _, page := range copy.Pages {
		if !reflect.DeepEqual(page.Strokes, one.Pages[0].Strokes) {
			t.Fatal("copy changed ink")
		}
	}
	if copy.Pages[1].Cells[0].Start != 5 {
		t.Fatal("copy lost UTF-16 source position")
	}
	full := strings.Repeat("正文\n", 4)
	copy, err = font.Layout(full+"\f"+full, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Pages) != 3 {
		t.Fatalf("boundary made a phantom page: %d", len(copy.Pages))
	}
	if copy.Pages[1].Text != full {
		t.Fatalf("explicit newlines changed: %q", copy.Pages[1].Text)
	}
}

func TestExplicitPageValidation(t *testing.T) {
	font, _ := EmbeddedFont()
	s := DefaultLayout()
	s.PageLayouts = []PageLayout{{LineCount: 0}}
	if _, err := font.Layout("a\fb", s); err == nil {
		t.Fatal("invalid grid accepted")
	}
	s.PageLayouts = nil
	if _, err := font.Layout(strings.Repeat("\f", 20001), s); err == nil {
		t.Fatal("unbounded pages accepted")
	}
}
