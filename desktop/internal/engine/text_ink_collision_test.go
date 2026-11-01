package engine

import "testing"

func TestCircledStepAndNextLineDoNotOverlapInsideEmptyCorners(t *testing.T) {
	s := compactDCLayout()
	s.FontID, s.LetterSpacing = FontHandwriting, 72.5
	doc, err := testFont(t).Layout("③\n看清", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages[0].Issues) != 0 || len(doc.Pages[0].Diagnostics) != 0 {
		t.Fatal("empty corners of ③ and 清 were treated as ink", doc.Pages[0].Issues, doc.Pages[0].Diagnostics)
	}
	if _, err := ExportAll(doc, DefaultWriting()); err != nil {
		t.Fatal("clear handwriting could not export at its original spacing", err)
	}
	s.HandwritingVariation, s.HandwritingSeed = 9, 2418718602
	doc, err = testFont(t).Layout("③\n看清", s)
	if err != nil || len(doc.Pages[0].Issues) != 0 || len(doc.Pages[0].Diagnostics) != 0 {
		t.Fatal("varied handwriting marked the empty corners as ink", err)
	}
}

func TestStrokeCollisionUsesVisiblePenWidthAndRoundEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name       string
		a, b, c, d Point
		touch      bool
	}{
		{"crossing", Point{0, 0}, Point{4, 4}, Point{0, 4}, Point{4, 0}, true},
		{"overlapping envelopes only", Point{0, 0}, Point{4, 4}, Point{0, 2}, Point{4, 6}, false},
		{"collinear gap", Point{0, 0}, Point{1, 0}, Point{2, 0}, Point{3, 0}, false},
		{"collinear overlap", Point{0, 0}, Point{2, 0}, Point{1, 0}, Point{3, 0}, true},
		{"pen edges touch", Point{0, 0}, Point{4, 0}, Point{0, bodyInkWidth}, Point{4, bodyInkWidth}, true},
		{"pen edges clear", Point{0, 0}, Point{4, 0}, Point{0, bodyInkWidth + .01}, Point{4, bodyInkWidth + .01}, false},
		{"round endpoints", Point{0, 0}, Point{1, 0}, Point{1.1, .1}, Point{1.1, 1}, true},
		{"degenerate point touches", Point{1, 0}, Point{1, 0}, Point{0, .2}, Point{2, .2}, true},
		{"degenerate point clears", Point{1, 0}, Point{1, 0}, Point{0, .3}, Point{2, .3}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if segmentsTouch(tc.a, tc.b, tc.c, tc.d) != tc.touch || segmentsTouch(tc.c, tc.d, tc.a, tc.b) != tc.touch {
				t.Fatal("collision disagrees with the visible strokes", tc)
			}
		})
	}
}

func TestCollisionTraceMatchesRenderedHandwriting(t *testing.T) {
	f, s := diagnosticFont(), diagnosticSettings()
	s.LineSpacing, s.HandwritingVariation = 100, 100
	runes := []rune("甲乙")
	for seed := range uint32(8) {
		s.HandwritingSeed = seed
		doc, err := f.Layout(string(runes), s)
		if err != nil {
			t.Fatal(err)
		}
		line := f.wrapTextLine(runes, 0, 0, s, 0, 10, nil)
		ink := textInk{}
		ink.addLine(f, runes, line, s, doc.Pages[0].Cells[0].Y, s.variation("body"))
		for i := range ink.active {
			trace := ink.active[i].trace()
			stroke := doc.Pages[0].Strokes[i]
			if len(trace) != 1 || trace[0].a != stroke[0] || trace[0].b != stroke[1] {
				t.Fatal("collision used different points from the preview/export", seed, i, trace, stroke)
			}
		}
	}
}
