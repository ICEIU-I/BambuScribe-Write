package engine

import "testing"

func diagnosticFont() *Font {
	return &Font{Units: 10, Glyphs: map[rune]Glyph{
		'甲': {Width: 6, Strokes: []Stroke{{{1, 1}, {5, 9}}}},
		'乙': {Width: 6, Strokes: []Stroke{{{1, 9}, {5, 1}}}},
	}}
}

func diagnosticSettings() LayoutSettings {
	s := DefaultLayout()
	s.PaperHeight, s.LineCount, s.FontSize, s.LineSpacing = 60, 6, 10, 50
	return s
}

func TestOverlapDiagnosticsLocateBothCollidingGlyphs(t *testing.T) {
	f, s := diagnosticFont(), diagnosticSettings()
	doc, err := f.Layout("甲\n乙", s)
	if err != nil {
		t.Fatal(err)
	}
	p := doc.Pages[0]
	if len(p.Diagnostics) != 2 || len(p.Issues) != 1 {
		t.Fatal("overlap did not identify the two affected glyphs", p.Diagnostics, p.Issues)
	}
	for i, diagnostic := range p.Diagnostics {
		want := RegionBounds{X: 3, Y: float64(1 + i*5), Width: 4, Height: 8}
		if diagnostic.RegionBounds != want || diagnostic.Code != "line-overlap" || diagnostic.Message != lineOverlapIssue {
			t.Fatal("diagnostic does not match the conflicting glyph's ink", diagnostic, want)
		}
	}
	// Markers are metadata: the two original strokes remain intact.
	if len(p.Strokes) != 2 || p.Strokes[0][0] != (Point{3, 1}) || p.Strokes[1][0] != (Point{3, 14}) {
		t.Fatal("diagnostics changed or added writing strokes", p.Strokes)
	}
	if _, err := ExportAll(doc, DefaultWriting()); err == nil {
		t.Fatal("localized diagnostics lost the existing export block")
	}
}

func TestOverlapDiagnosticsClearWhenSpacingIsFixedAndStayOnTheirPage(t *testing.T) {
	f, s := diagnosticFont(), diagnosticSettings()
	doc, err := f.Layout("甲\f甲\n乙", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Diagnostics) != 0 || len(doc.Pages[1].Diagnostics) != 2 || doc.Pages[1].Diagnostics[0].Y != 1 {
		t.Fatal("diagnostics leaked across pages or used document coordinates")
	}
	s.LineSpacing = 100
	doc, err = f.Layout("甲\f甲\n乙", s)
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range doc.Pages {
		if len(page.Diagnostics) != 0 || len(page.Issues) != 0 {
			t.Fatal("fixed spacing kept stale diagnostics")
		}
	}
	if _, err := ExportAll(doc, DefaultWriting()); err != nil {
		t.Fatal("corrected spacing could not export", err)
	}
	doc, err = f.Layout("甲\n     乙", diagnosticSettings())
	if err != nil || len(doc.Pages[0].Diagnostics) != 0 {
		t.Fatal("horizontally disjoint glyphs were marked", err)
	}
}

func TestRejectedHandwritingSpacingTrialDoesNotLeaveDiagnostics(t *testing.T) {
	f := &Font{Units: 10, Glyphs: map[rune]Glyph{'甲': {Width: 10, Strokes: []Stroke{{{0, 0}, {10, 2}}}}}}
	s := DefaultLayout()
	s.FontSize, s.HandwritingVariation = 10, 100
	line := f.wrapTextLine([]rune("甲"), 0, 0, s, 5, 10, nil)
	for seed := range uint32(100) {
		s.HandwritingSeed = seed
		v := s.variation("body")
		if 5+v.spaceOffset(0, true, 10, 5, 8) >= 4 {
			continue
		}
		ink := textInk{active: []textInkGlyph{{RegionBounds: RegionBounds{X: 0, Y: 0, Width: 10, Height: 4}}}}
		y, fits := f.placeVariedTextLine(&ink, []rune("甲"), line, s, 5, 10, nil, v)
		if !fits || y != 5 || len(ink.overlaps) != 0 {
			t.Fatal("discarded trial produced a stale overlap marker", y, fits, ink.overlaps)
		}
		return
	}
	t.Fatal("fixture did not exercise the rejected spacing trial")
}
