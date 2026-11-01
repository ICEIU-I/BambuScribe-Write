package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestDCFontCoversExistingFontAndLessons(t *testing.T) {
	dc, err := BuiltinFont(FontDC)
	if err != nil {
		t.Fatal(err)
	}
	base := testFont(t)
	for r := range base.Glyphs {
		if len(dc.Glyphs[r].Strokes) == 0 {
			t.Errorf("DC font lost existing character %q", r)
		}
	}
	corpus, err := os.ReadFile("testdata/lesson-characters.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range string(corpus) {
		// The combining overbar is attached by the table/formula renderer.
		if unicode.IsSpace(r) || r == '\u0305' {
			continue
		}
		if len(dc.Glyphs[r].Strokes) == 0 {
			t.Errorf("DC font cannot draw lesson character %q", r)
		}
	}
	for r, g := range dc.Glyphs {
		if !finite(g.Width) || g.Width < 0 || g.Width > dc.Units {
			t.Errorf("invalid width for %q", r)
		}
		for _, s := range g.Strokes {
			if len(s) < 2 {
				t.Errorf("invalid stroke for %q", r)
			}
			for _, p := range s {
				if !finite(p.X) || !finite(p.Y) || p.X < 0 || p.Y < 0 || p.X > dc.Units || p.Y > dc.Units {
					t.Errorf("%q leaves its writing cell: %+v", r, p)
				}
			}
		}
	}
	t.Logf("DC=%d, original=%d, lesson characters=%d", len(dc.Glyphs), len(base.Glyphs), len([]rune(string(corpus))))
}

func TestDCUsesOwnHandwrittenNumbers(t *testing.T) {
	dc, err := BuiltinFont(FontDC)
	if err != nil {
		t.Fatal(err)
	}
	base := testFont(t)
	for _, r := range "永实验0123¹₁½①⑩" {
		if reflect.DeepEqual(dc.Glyphs[r], base.Glyphs[r]) {
			t.Errorf("DC character %q reused the old font", r)
		}
	}
	for _, r := range "¹₁" {
		g := dc.Glyphs[r]
		a, b, c, d, ok := inkBounds(g.Strokes)
		x, y, z, w, _ := inkBounds(dc.Glyphs['1'].Strokes)
		if !ok || math.Abs((c-a)/(d-b)-(z-x)/(w-y)) > 1e-8 {
			t.Fatalf("%q distorted the original digit", r)
		}
	}
}

func TestFontSelectionRewritesImageLabels(t *testing.T) {
	base := testFont(t)
	dc, _ := BuiltinFont(FontDC)
	s := DefaultLayout()
	s.FontID = FontDC
	im := ImageAsset{ID: "diagram", Data: testImageData(t), X: 10, Y: 40, Width: 60, Height: 40,
		Diagram:     &diagram.Scene{Width: 120, Height: 80, Labels: []diagram.Label{{Text: "实验 U_{H}", X: 10, Y: 10, Width: 100, Height: 30}}},
		TextRegions: []ImageTextRegion{{ID: "manual", Source: "manual", Mode: "handwriting", Text: "霍尔", X: .1, Y: .6, Width: .7, Height: .3}}}
	want, err := diagramImageStrokes(dc, im, s)
	if err != nil {
		t.Fatal(err)
	}
	d, err := base.LayoutWithImages("", s, []ImageAsset{im})
	if err != nil || !reflect.DeepEqual(d.Pages[0].Images[0].Strokes, want.Strokes) {
		t.Fatal("diagram/manual image labels used the old font", err)
	}
}

func TestSelectedFontRendersBodyObjectsAndExport(t *testing.T) {
	base := testFont(t)
	s := DefaultLayout()
	s.FontSize, s.FontID = 6, FontDC
	table := TableAsset{ID: "formula", X: 10, Y: 40, ColumnWidths: []float64{140}, RowHeights: []float64{20}, Cells: []TableCell{{RowSpan: 1, ColSpan: 1, Text: "U̅_{H}=R_{H}·I_{S}B/d 10⁻¹⁹ ①"}}, FontSize: 5, Padding: 1, Style: "grid", Align: "left", VerticalAlign: "middle"}
	d, err := base.LayoutWithObjects("实验霍尔效应", s, nil, []TableAsset{table})
	if err != nil || len(d.Missing) != 0 || d.FontName != "DC体" || len(d.Pages[0].Issues) != 0 {
		t.Fatalf("selected font render failed: %+v, %v", d, err)
	}
	dc, _ := BuiltinFont(FontDC)
	want := dc.renderTable(table, s)
	if !reflect.DeepEqual(want, d.Pages[0].Tables[0]) {
		t.Fatal("table measurement and rendering used another font")
	}
	if _, err := ExportPage(d, 0, DefaultWriting()); err != nil {
		t.Fatal(err)
	}
	s.FontID = ""
	old, err := base.LayoutWithObjects("实验霍尔效应", s, nil, []TableAsset{table})
	if err != nil || old.ID == d.ID || old.FontName != "教案专用" || reflect.DeepEqual(old.Pages[0].Strokes, d.Pages[0].Strokes) {
		t.Fatal("font switch did not isolate preview/export cache", err)
	}
	if base.Name != "教案专用" || base.ID != FontHandwriting {
		t.Fatal("font switch mutated the shared original font")
	}
	// Explicit pages and empty documents need distinct cache IDs as well.
	for _, text := range []string{"", "实验\f实验"} {
		a, _ := base.Layout(text, s)
		s.FontID = FontDC
		b, err := base.Layout(text, s)
		if err != nil || a.ID == b.ID || b.FontName != "DC体" {
			t.Fatal("explicit/empty page reused original font cache", err)
		}
		s.FontID = ""
	}
}

func TestUnknownFontIsRejected(t *testing.T) {
	s := DefaultLayout()
	s.FontID = "missing"
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "重新选择字体") {
		t.Fatal("unknown font should provide an actionable error", err)
	}
	if _, err := testFont(t).Layout("实验", s); err == nil {
		t.Fatal("unknown font silently used original handwriting")
	}
}
