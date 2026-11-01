package engine

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"
)

func TestTableRulesRemainStraightAndMergedCellsOpen(t *testing.T) {
	const w, h = 220, 140
	mask := make([]bool, w*h)
	// Simulate scan drift, small dropouts and merged cells in the first column.
	for _, y := range []int{12, 40, 70, 100, 126} {
		left := 10
		if y == 70 || y == 100 {
			left = 60
		}
		for x := left; x <= 209; x++ {
			if x == 115 || x == 116 {
				continue
			}
			dy := 0
			if x > 120 {
				dy = 1
			}
			mask[(y+dy)*w+x] = true
		}
	}
	for _, x := range []int{60, 110, 160} {
		for y := 12; y <= 127; y++ {
			if y == 82 || y == 83 {
				continue
			}
			dx := 0
			if y > 85 {
				dx = 1
			}
			mask[y*w+x+dx] = true
		}
	}
	rules := rectifiedTableLines(mask, w, h)
	if len(rules) != 8 {
		t.Fatalf("expected 5 horizontal and 3 vertical rules, got %d", len(rules))
	}
	for _, s := range rules {
		if len(s) != 2 {
			t.Fatal("rules must be single segments")
		}
		a, b := s[0], s[1]
		if a.Y == b.Y {
			if a.Y > 60 && a.Y < 110 && a.X < 59 {
				t.Fatal("filled an intentional merged-cell opening")
			}
			if b.X < 208 {
				t.Fatal("lost right-hand rule end")
			}
		} else if a.X != b.X {
			t.Fatal("rule is not straight")
		}
	}
	remaining := 0
	for _, v := range mask {
		if v {
			remaining++
		}
	}
	if remaining > 0 {
		t.Fatalf("left %d duplicate raster pixels", remaining)
	}
}

func TestTextMaskDoesNotCutDiagramPassingThroughOCRBox(t *testing.T) {
	const w, h = 100, 50
	mask := make([]bool, w*h)
	for x := 0; x < w; x++ {
		mask[25*w+x] = true
	}
	for y := 10; y < 15; y++ {
		for x := 40; x < 46; x++ {
			mask[y*w+x] = true
		}
	}
	eraseImageText(mask, w, h, []OCRBox{{Text: "U", X: .35, Y: .15, W: .2, H: .5}})
	for x := 0; x < w; x++ {
		if !mask[25*w+x] {
			t.Fatal("OCR erased a curve/axis")
		}
	}
	if mask[12*w+43] {
		t.Fatal("source text left behind")
	}
}

func TestSkeletonLoopsCornersAndEdgeInk(t *testing.T) {
	const w, h = 30, 30
	mask := make([]bool, w*h)
	for y := 5; y <= 24; y++ {
		mask[y*w+5] = true
		mask[y*w+24] = true
	}
	for x := 5; x <= 24; x++ {
		mask[5*w+x] = true
		mask[24*w+x] = true
	}
	paths := traceImageSkeleton(thinImageMask(mask, w, h), w, h, func(x, y int) Point { return Point{float64(x), float64(y)} })
	if len(paths) != 1 || paths[0][0] != paths[0][len(paths[0])-1] {
		t.Fatal("loop lost its closing edge or split at corners")
	}
	if math.Abs(strokeLength(paths[0])-76) > 1e-8 {
		t.Fatal("retraced/cut rectangle edges")
	}
	edge := make([]bool, w*h)
	for x := 0; x < w; x++ {
		edge[x] = true
		edge[w+x] = true
		edge[2*w+x] = true
	}
	paths = traceImageSkeleton(thinImageMask(edge, w, h), w, h, func(x, y int) Point { return Point{float64(x), float64(y)} })
	if len(paths) != 1 {
		t.Fatalf("image edge should be one centreline, got %d", len(paths))
	}
}

func TestSmoothingRemovesRasterStepsWithoutRoundingRightAngle(t *testing.T) {
	input := Stroke{}
	for x := 0; x <= 100; x++ {
		input = append(input, Point{float64(x), math.Round(float64(x) * .37)})
	}
	got := smoothImageStroke(input)
	if len(got) > len(input)/3 {
		t.Fatal("pixel staircase retained")
	}
	if got[0] != input[0] || got[len(got)-1] != input[len(input)-1] {
		t.Fatal("endpoints moved")
	}
	for _, p := range got {
		if pointLineDistance(p, input[0], input[len(input)-1]) > .7 {
			t.Fatal("curve fitting drift")
		}
	}
	corner := Stroke{{0, 0}, {5, 0}, {10, 0}, {10, 5}, {10, 10}}
	smoothed := smoothImageStroke(corner)
	found := false
	for _, p := range smoothed {
		if p == (Point{10, 0}) {
			found = true
		}
	}
	if !found {
		t.Fatal("sharp corner rounded")
	}
}

func TestDashesUseEqualArcLengthAroundBends(t *testing.T) {
	path := Stroke{}
	for i := 0; i <= 180; i++ {
		a := float64(i) * math.Pi / 180
		path = append(path, Point{50 * math.Cos(a), 50 * math.Sin(a)})
	}
	dashes := dashImageStroke(path, 8, 4)
	for i, s := range dashes {
		if i < len(dashes)-1 && math.Abs(strokeLength(s)-8) > 1e-6 {
			t.Fatal("unequal dash length")
		}
		if i > 0 {
			gap := distance(dashes[i-1][len(dashes[i-1])-1], s[0])
			if math.Abs(gap-4) > .01 {
				t.Fatal("unequal gap on curve")
			}
		}
	}
}

func TestNormalizeDashesPreservesSolidLinesAndSeparateParallelChains(t *testing.T) {
	strokes := []Stroke{{{0, 80}, {140, 80}}}
	for _, y := range []float64{0, 12} {
		for i := 0; i < 9; i++ {
			x := float64(i * 13)
			length := float64(5 + i%4)
			strokes = append(strokes, Stroke{{x, y}, {x + length, y}})
		}
	}
	got := normalizeDashedStrokes(strokes)
	solid := 0
	for _, s := range got {
		if s[0].Y == 80 {
			solid++
			if strokeLength(s) != 140 {
				t.Fatal("solid rule dashed")
			}
		} else if s[0].Y != s[len(s)-1].Y {
			t.Fatal("joined parallel chains")
		}
	}
	if solid != 1 {
		t.Fatal("lost solid rule")
	}
	for _, y := range []float64{0, 12} {
		previous := -1.
		length := -1.
		gap := -1.
		count := 0
		for _, s := range got {
			if s[0].Y != y {
				continue
			}
			n := strokeLength(s)
			if length < 0 {
				length = n
			}
			if n > length+.01 {
				t.Fatal("unequal dash lengths")
			}
			if previous >= 0 {
				g := s[0].X - previous
				if gap < 0 {
					gap = g
				}
				if math.Abs(g-gap) > .01 {
					t.Fatal("unequal gaps")
				}
			}
			previous = s[len(s)-1].X
			count++
		}
		if count < 5 {
			t.Fatal("lost dashed chain")
		}
	}
}

func TestHandwrittenLabelDoesNotOverlapAndUnknownGlyphIsKept(t *testing.T) {
	f := &Font{Units: 10, Glyphs: map[rune]Glyph{'A': {Width: 2, Strokes: []Stroke{{{0, 0}, {10, 10}}}}}}
	a := ImageAsset{Width: 60, Height: 40}
	box := OCRBox{Text: "AA", W: 1, H: .25}
	s := handwritingTextStrokes(f, box, a, DefaultLayout(), 100, 100)
	if len(s) != 2 || s[0][1].X >= s[1][0].X {
		t.Fatal("glyph ink overlaps next letter")
	}
	box.Text = "A🙂"
	if len(handwritingTextStrokes(f, box, a, DefaultLayout(), 100, 100)) != 0 {
		t.Fatal("unsupported label partially overwritten")
	}
}

func TestImageQualityPreviewAndGCodeHaveIdenticalPenPaths(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 140))
	for y := 0; y < 140; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.White)
		}
	}
	for _, y := range []int{10, 50, 90, 130} {
		for x := 10; x <= 190; x++ {
			img.Set(x, y, color.Black)
		}
	}
	for _, x := range []int{10, 70, 130, 190} {
		for y := 10; y <= 130; y++ {
			img.Set(x, y, color.Black)
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	f := testFont(t)
	asset := ImageAsset{ID: "table", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes()), X: 10, Y: 20, Width: 120, Height: 84}
	doc, err := f.LayoutWithImages("表格", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	p := doc.Pages[0]
	if len(p.Strokes) != len(p.TextStrokes)+len(p.Images[0].Strokes) {
		t.Fatal("preview duplicates image paths")
	}
	data, err := ExportPage(doc, 0, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	expected, err := GenerateGCode(p, doc.Settings, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("G-code differs from preview")
	}
	if strings.Count(expected, "G1 Z3.400") != len(p.Strokes) {
		t.Fatal("lost pen lifts")
	}
}

// A ruled figure: three horizontal rules, two vertical ones, all long enough to
// be recognised as rules. Sizes are chosen so `rectifiedTableLines` accepts it
// (minimum run = max(30, major/5)) while the details below stay short enough
// not to become rules themselves.
func ruledMask(width, height int) []bool {
	mask := make([]bool, width*height)
	line := func(x0, y0, x1, y1 int) {
		steps := max(abs(x1-x0), abs(y1-y0)) + 1
		for i := 0; i <= steps; i++ {
			x := x0 + (x1-x0)*i/steps
			y := y0 + (y1-y0)*i/steps
			mask[y*width+x] = true
		}
	}
	for _, y := range []int{20, 100, 180} {
		line(30, y, 370, y)
	}
	for _, x := range []int{100, 300} {
		line(x, 20, x, 180)
	}
	return mask
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Cutting a rule's ink out must not cut the line crossing it: a wire lost 2.9mm
// where a rule passed over it, with nothing left to rejoin the halves.
func TestRuleInkRemovalKeepsCrossingLineWhole(t *testing.T) {
	const w, h = 400, 200
	mask := ruledMask(w, h)
	wire := 60
	for x := 40; x <= 115; x++ {
		mask[wire*w+x] = true
		mask[(wire+1)*w+x] = true
	}
	if rules := rectifiedTableLines(mask, w, h); len(rules) != 5 {
		t.Fatalf("expected 3 horizontal and 2 vertical rules, got %d", len(rules))
	}
	for x := 40; x <= 115; x++ {
		if !mask[wire*w+x] {
			t.Fatalf("wire cut at column %d where the rule crosses", x)
		}
	}
	for a := 96; a <= 104; a++ {
		if mask[wire*w+a] && (a < 95 || a > 105) {
			t.Fatal("rule ink left beside the crossing")
		}
	}
	if mask[wire*w+300] {
		t.Fatal("rule ink kept where nothing crosses it")
	}
}

// Ink near a rule is the rule itself, removed for the straight stroke that
// replaces it. Only lines passing right through are held back.
func TestRuleInkRemovalStillClearsUncrossedRules(t *testing.T) {
	const w, h = 400, 200
	mask := ruledMask(w, h)
	rectifiedTableLines(mask, w, h)
	for y := 20; y <= 180; y++ {
		for _, x := range []int{100, 300} {
			for a := x - 3; a <= x+3; a++ {
				if y > 96 && y < 104 {
					continue // the horizontal rule's own band, kept where they cross
				}
				if mask[y*w+a] {
					t.Fatalf("rule ink left at (%d,%d)", a, y)
				}
			}
		}
	}
}

// Bar three of a ground symbol survived as a fraction of a millimetre, because
// the vertical rule through it deleted each bar's middle.
func TestRuleRemovalKeepsGroundSymbolBars(t *testing.T) {
	const w, h = 400, 200
	mask := ruledMask(w, h)
	for y := 20; y <= 180; y++ {
		mask[y*w+200] = true
	}
	bars := []struct{ y, from, to int }{{140, 180, 220}, {150, 185, 215}, {160, 190, 210}}
	for _, b := range bars {
		for x := b.from; x <= b.to; x++ {
			mask[b.y*w+x] = true
			mask[(b.y+1)*w+x] = true
		}
	}
	rectifiedTableLines(mask, w, h)
	for _, b := range bars {
		for x := b.from; x <= b.to; x++ {
			if !mask[b.y*w+x] {
				t.Fatalf("bar at y=%d cut at column %d", b.y, x)
			}
		}
	}
}

// A box edge and the rule it meets are traced apart, so the edge stops short of
// the rule's axis and the corner is left open.
func TestSnapMovesStrokeEndsOntoRuleAxis(t *testing.T) {
	rules := []Stroke{{{100, 20}, {100, 180}}, {{30, 140}, {370, 140}}}
	paths := []Stroke{
		{{60, 80}, {94, 80}},     // stops 6px short of the vertical rule
		{{300, 120}, {300, 134}}, // ...and this one short of the horizontal rule
	}
	got := snapStrokesToRules(paths, rules)
	if got[0][1] != (Point{100, 80}) {
		t.Fatalf("end not pulled onto the vertical rule: %v", got[0][1])
	}
	if got[1][1] != (Point{300, 140}) {
		t.Fatalf("end not pulled onto the horizontal rule: %v", got[1][1])
	}
	away := snapStrokesToRules([]Stroke{{{120, 80}, {160, 80}}}, rules)
	if away[0][0] != (Point{120, 80}) || away[0][1] != (Point{160, 80}) {
		t.Fatal("pulled an end that was moving away from the rule")
	}
}

// Branches of one junction rarely end on the same pixel after thinning, so
// exact matching left them as separate strokes with a gap between them.
func TestJunctionBranchesMergeAcrossAFewPixels(t *testing.T) {
	joined := joinImageJunctions([]Stroke{{{0, 0}, {50, 0}}, {{52, 0}, {100, 0}}})
	if len(joined) != 1 || len(joined[0]) < 3 {
		t.Fatalf("collinear branches were not joined: %d strokes", len(joined))
	}
	if joined[0][0] != (Point{0, 0}) || joined[0][len(joined[0])-1] != (Point{100, 0}) {
		t.Fatal("joined path lost an end")
	}
	// A corner is a pen turn, not a break: the tangents must still oppose.
	corner := joinImageJunctions([]Stroke{{{0, 80}, {50, 80}}, {{52, 80}, {52, 130}}})
	if len(corner) != 2 {
		t.Fatal("joined two strokes that only meet at a corner")
	}
}

// Specks left by thinning a solid mark make the pen tap dots. A short stroke
// lying against other ink is a real detail and stays.
func TestIsolatedSpecksDroppedBesideInkKept(t *testing.T) {
	strokes := []Stroke{
		{{0, 0}, {60, 0}},
		{{100, 100}, {101.5, 100}},
		{{30, 3}, {31.5, 3}},
	}
	got := dropIsolatedSpecks(strokes, 3, 4)
	if len(got) != 2 {
		t.Fatalf("expected the loose speck dropped, got %d strokes", len(got))
	}
	for _, s := range got {
		if s[0] == (Point{100, 100}) {
			t.Fatal("dropped specks but kept the far one")
		}
	}
}
