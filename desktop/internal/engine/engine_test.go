package engine

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"
	"testing"
)

func testImageData(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 12, 12))
	for y := 2; y < 10; y++ {
		for x := 2; x < 10; x++ {
			img.Set(x, y, color.Black)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

func testFont(t *testing.T) *Font {
	t.Helper()
	f, err := EmbeddedFont()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestEmbeddedFontAndSupplement(t *testing.T) {
	f := testFont(t)
	if len(f.Glyphs) < 6848 {
		t.Fatal("font lost characters")
	}
	for _, r := range "你好炽写Aa012，。！？；：（）“”、.?![]×μµ$¥￥＄" {
		if len(f.Glyphs[r].Strokes) == 0 {
			t.Errorf("missing %c", r)
		}
	}
	if len(f.Glyphs['你'].Strokes) != 6 {
		t.Fatal("original stroke boundaries changed")
	}
}

func TestSupplementedSymbolsDoNotBecomeMissingGlyphs(t *testing.T) {
	f := testFont(t)
	d, err := f.Layout("100×μA，100×µA $¥￥＄", DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Missing) != 0 || len(d.Pages[0].Missing) != 0 {
		t.Fatalf("supplemented symbols reported missing: document=%v page=%v", d.Missing, d.Pages[0].Missing)
	}
}

func TestPunctuationUsesCompactGlyphAdvance(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.LetterGap = 0
	for _, r := range "、。〈〉《》【】！？（），“”‘’—…＋－／＜＝＞＼～" {
		advance := f.advance(r, s)
		if advance <= 0 || advance >= s.FontSize {
			t.Fatalf("punctuation %q should use its compact glyph advance, got %.3f", r, advance)
		}
	}
	wide := f.advance('你', s)
	if f.advance('“', s) >= wide || f.advance('”', s) >= wide {
		t.Fatal("paired quotes should be narrower than a CJK character")
	}
}

func TestCommonScienceSymbolsAndSerialNumbers(t *testing.T) {
	f := testFont(t)
	text := "ΑΒΓΔΩαβγδεζηθλμνξπρστφχψω Ω℧ÅÅ∆∇∂∑∏∫√∞≈≠≤≥∝∀∃¬∧∨∈∉∅∪∩⊂⊃⊆⊇⊕⊗≪≫∴∵ℏℓ℃℉№※†‡−－＋＝＜＞／＼～′″‰‱©®™㎜㎝㎞㎏㎎㎍㎚㎛㎲㎳㎐㎑㎒㎓㎔㏀㏁㏂㏃㏄㏅㏈㏉㏋㏌㏏㏒㏓㏔㏕㏖㏗㏙㏚㏛㏜㏝←↑→↓↔↕↖↗↘↙⇒⇐⇔⁰¹²³⁴⁵⁶⁷⁸⁹₀₁₂₃₄₅₆₇₈₉⁺⁻⁼₊₋₌½⅓⅔¼¾⅕⅖⅗⅘⅙⅚⅐⅛⅜⅝⅞⓪①②③④⑤⑥⑦⑧⑨⑩⑪⑫⑬⑭⑮⑯⑰⑱⑲⑳㉑㉒㉓㉔㉕㉖㉗㉘㉙㉚㉛⑴⑵⑶⑷⑸⑹⑺⑻⑼⑽⑾⑿⒀⒁⒂⒃⒄⒅⒆⒇"
	for _, r := range text {
		if r == ' ' {
			continue
		}
		g, ok := f.Glyphs[r]
		if !ok || len(g.Strokes) == 0 {
			t.Errorf("missing supplemental glyph %q", r)
			continue
		}
		for _, stroke := range g.Strokes {
			for _, p := range stroke {
				if p.X < 0 || p.X > f.Units || p.Y < 0 || p.Y > f.Units {
					t.Errorf("glyph %q leaves cell: %+v", r, p)
				}
			}
		}
	}
	d, err := f.Layout(text, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Missing) != 0 {
		t.Fatalf("common science symbols reported missing: %v", d.Missing)
	}
	if len(f.Glyphs['①'].Strokes) != 2 || len(f.Glyphs['⑩'].Strokes) != 3 || len(f.Glyphs['⑴'].Strokes) != 3 {
		t.Fatal("serial glyphs are not composed from the handwritten circle/parentheses and source digits")
	}
}

// Composed digits have to keep the proportions of the glyph they reuse. Fitting
// both axes to the target box stretched them instead: this font's 1 is a narrow
// stroke beside a wide 2, so a superscript ¹ came out as a slash, and the 1 of ½
// with it. Shapes that are meant to be reshaped — the ring of ①, the parentheses
// of ⑴ — are excluded here and pinned separately below.
func TestComposedDigitsKeepSourceProportions(t *testing.T) {
	f := testFont(t)
	aspect := func(strokes []Stroke) float64 {
		minX, minY, maxX, maxY := 1e9, 1e9, -1e9, -1e9
		for _, s := range strokes {
			for _, p := range s {
				minX, minY = min(minX, p.X), min(minY, p.Y)
				maxX, maxY = max(maxX, p.X), max(maxY, p.Y)
			}
		}
		if maxY-minY <= 1e-9 {
			return 0
		}
		return (maxX - minX) / (maxY - minY)
	}
	sameAspect := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 0.02*want+0.02 {
			t.Errorf("%s: aspect %.3f, source glyph is %.3f", name, got, want)
		}
	}
	for i, r := range []rune("⁰¹²³⁴⁵⁶⁷⁸⁹") {
		digits := f.Glyphs[r].Strokes
		sameAspect("superscript "+string(r), aspect(digits), aspect(f.Glyphs[rune('0'+i)].Strokes))
	}
	for i, r := range []rune("₀₁₂₃₄₅₆₇₈₉") {
		digits := f.Glyphs[r].Strokes
		sameAspect("subscript "+string(r), aspect(digits), aspect(f.Glyphs[rune('0'+i)].Strokes))
	}
	for _, r := range []rune("①②③④⑤⑥⑦⑧⑨") {
		// Stroke 0 is the ring; the digit is what follows it.
		digit := rune('0' + int(r-0x245f))
		sameAspect("circled "+string(r), aspect(f.Glyphs[r].Strokes[1:]), aspect(f.Glyphs[digit].Strokes))
	}
	for r, pair := range map[rune][2]int{'½': {1, 2}, '⅓': {1, 3}, '⅞': {7, 8}, '⅘': {4, 5}} {
		// Stroke 0 is the fraction bar; the digits follow it, and a digit may
		// be written with more than one stroke, so the split follows the source.
		strokes := f.Glyphs[r].Strokes
		top, bottom := f.Glyphs[rune('0'+pair[0])].Strokes, f.Glyphs[rune('0'+pair[1])].Strokes
		split := 1 + len(top)
		sameAspect("fraction top "+string(r), aspect(strokes[1:split]), aspect(top))
		sameAspect("fraction bottom "+string(r), aspect(strokes[split:]), aspect(bottom))
	}
	// The ring and the parentheses are the shapes that are stretched on purpose:
	// a 0 pulled into a square, and brackets standing the symbol's full height.
	ring := aspect(f.Glyphs['①'].Strokes[:1])
	if math.Abs(ring-1) > 0.15 {
		t.Errorf("circled ring is no longer round: aspect %.3f", ring)
	}
	if bracket := aspect(f.Glyphs['⑴'].Strokes[:1]); bracket > 0.35 {
		t.Errorf("composition bracket is no longer stretched vertically: aspect %.3f", bracket)
	}
}

func TestPaginationPreservesTextAndBounds(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	// Compare explicit legacy millimeter sizes independently of UI defaults.
	s.FontSize = 6
	text := strings.Repeat(strings.Repeat("你好，世界！这是写字测试。Hello 0123", 3)+"\n\n", 90)
	d, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages) < 2 {
		t.Fatal("expected pagination")
	}
	var recovered strings.Builder
	for _, p := range d.Pages {
		recovered.WriteString(p.Text)
		for _, stroke := range p.Strokes {
			for _, q := range stroke {
				if q.X < s.MarginLeft-1e-8 || q.X > s.PaperWidth-s.MarginRight+1e-8 || q.Y < s.MarginTop-1e-8 || q.Y > s.PaperHeight-s.MarginBottom+1e-8 {
					t.Fatalf("out of page: %+v", q)
				}
			}
		}
	}
	if recovered.String() != text {
		t.Fatal("pagination dropped or duplicated text")
	}
	d2, _ := f.Layout(text, s)
	if d.ID != d2.ID {
		t.Fatal("unstable document version")
	}
	s.FontSize = 9
	d3, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	if len(d3.Pages) <= len(d.Pages) || d3.ID == d.ID {
		t.Fatal("font size must affect pagination/version")
	}
}

func TestPunctuationMovesWithPrecedingCharacter(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.MarginLeft = 1
	s.MarginRight = 1
	s.PaperWidth = 3*s.FontSize + s.MarginLeft + s.MarginRight
	d, err := f.Layout("你好你，好", s)
	if err != nil {
		t.Fatal(err)
	}
	// Third character moves to the next line with its comma.
	thirdStart := len(f.Glyphs['你'].Strokes) + len(f.Glyphs['好'].Strokes)
	if d.Pages[0].Strokes[thirdStart][0].Y < s.MarginTop+s.LineHeight {
		t.Fatal("comma orphaned at line start")
	}
}

func TestMissingGlyphStopsOnlyAffectedPage(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.PaperHeight = 20
	s.LineCount = 1
	d, err := f.Layout("你好\n🙂", s)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages) != 2 || len(d.Pages[1].Missing) != 1 || len(d.Missing) != 1 {
		t.Fatalf("missing glyph metadata: %+v", d.Missing)
	}
	if _, err = ExportPage(d, 0, testWriting()); err != nil {
		t.Fatal(err)
	}
	if _, err = ExportPage(d, 1, testWriting()); err == nil {
		t.Fatal("export must reject missing glyph")
	}
}

func TestAbsoluteStrokesTrackPreviewWithoutDrift(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	d, _ := f.Layout(strings.Repeat("你好。", 40), s)
	gcode, err := GenerateGCode(d.Pages[0], s, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(gcode, "G1 Z3.400") != len(d.Pages[0].Strokes) || strings.Count(gcode, "G1 Z7.000") != len(d.Pages[0].Strokes)+1 || strings.Count(gcode, "G1 Z50.000") != 1 {
		t.Fatal("lost pen boundaries")
	}
	var expected []Point
	for _, stroke := range d.Pages[0].Strokes {
		for _, p := range stroke {
			expected = append(expected, Point{p.X, s.PaperHeight - p.Y})
		}
	}
	expected = append(expected, Point{})
	x, y, z, index := 0.0, 0.0, 0.0, 0
	for _, line := range strings.Split(gcode, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || (fields[0] != "G0" && fields[0] != "G1") {
			continue
		}
		xy := false
		for _, v := range fields[1:] {
			n, e := strconv.ParseFloat(v[1:], 64)
			if e != nil {
				t.Fatal(e)
			}
			switch v[0] {
			case 'X':
				x = n
				xy = true
			case 'Y':
				y = n
				xy = true
			case 'Z':
				z = n
			}
		}
		if xy {
			want := expected[index]
			if math.Abs(x-want.X) > 0.000501 || math.Abs(y-want.Y) > 0.000501 {
				t.Fatalf("preview drift at %d: got %.6f,%.6f want %+v", index, x, y, want)
			}
			index++
		}
	}
	if index != len(expected) || math.Abs(x) > 1e-8 || math.Abs(y) > 1e-8 || z != 50 {
		t.Fatal("final position should park at XY zero with pen raised to Z50")
	}
	if strings.Contains(gcode, "G91\n") || !strings.Contains(gcode, "G90\n") {
		t.Fatal("writing must use absolute coordinates")
	}
	for _, bad := range []string{"G28", "G92", "G1 Z90", " E1"} {
		if strings.Contains(gcode, bad) {
			t.Fatalf("unexpected command %s", bad)
		}
	}
}

func TestRelativeBoundsAndInvalidInputs(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	page := Page{Number: 1, Strokes: []Stroke{{{X: s.MarginLeft, Y: s.MarginTop}, {X: s.MarginLeft + 257, Y: s.MarginTop}}}}
	if _, err := GenerateGCode(page, s, testWriting()); err == nil {
		t.Fatal("accepted unreachable relative span")
	}
	page.Strokes[0][1].X = math.NaN()
	if _, err := GenerateGCode(page, s, testWriting()); err == nil {
		t.Fatal("accepted NaN")
	}
	s.FontSize = math.Inf(1)
	if _, err := f.Layout("你", s); err == nil {
		t.Fatal("accepted infinity")
	}
	if _, err := f.Layout(strings.Repeat("你", 20001), DefaultLayout()); err == nil {
		t.Fatal("missing document limit")
	}
}

func TestExportContainsExactGeneratedGCode(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	d, _ := f.Layout("测试。", s)
	data, err := ExportPage(d, 0, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	expected, err := GenerateGCode(d.Pages[0], s, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatal("export differs from generated paths")
	}
}

func TestScientificSymbolsCanExport(t *testing.T) {
	f := testFont(t)
	d, err := f.Layout("× μ µ ℃ ± ÷", DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Missing) != 0 {
		t.Fatalf("scientific symbols missing: %v", d.Missing)
	}
	if _, err := ExportPage(d, 0, testWriting()); err != nil {
		t.Fatal(err)
	}
}

func TestImageIsMergedIntoPageStrokesAndGCode(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	d, err := f.LayoutWithImages("图片：", s, []ImageAsset{{ID: "img-1", Page: 0, Data: testImageData(t), X: 30, Y: 40, Width: 40, Height: 40, Mode: "line"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Pages[0].Images) != 1 || len(d.Pages[0].Images[0].Strokes) == 0 || len(d.Pages[0].Strokes) == 0 {
		t.Fatal("image was not added to the page")
	}
	if _, err = ExportPage(d, 0, testWriting()); err != nil {
		t.Fatal(err)
	}
}

func TestOCRTextIsRedrawnWithHandwritingGlyphs(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	strokes := handwritingTextStrokes(f, OCRBox{Text: "你好", X: 0.1, Y: 0.2, W: 0.4, H: 0.12}, ImageAsset{X: 10, Y: 20, Width: 100, Height: 80}, s, 400, 320)
	if len(strokes) == 0 {
		t.Fatal("OCR text was not converted to handwriting strokes")
	}
}

func TestImageSkeletonKeepsOneCenterline(t *testing.T) {
	mask := make([]bool, 30*12)
	for y := 4; y <= 7; y++ {
		for x := 3; x < 27; x++ {
			mask[y*30+x] = true
		}
	}
	strokes := traceImageSkeleton(thinImageMask(mask, 30, 12), 30, 12, func(x, y int) Point {
		return Point{X: float64(x), Y: float64(y)}
	})
	if len(strokes) != 1 {
		t.Fatalf("thick line should produce one centerline, got %d strokes", len(strokes))
	}
}

func TestImageTextRegionOverrideSurvivesPreview(t *testing.T) {
	f := testFont(t)
	asset := ImageAsset{
		ID: "ocr-override", Page: 0, Data: testImageData(t), X: 20, Y: 20, Width: 50, Height: 50,
		TextRegions: []ImageTextRegion{{ID: "label-1", Text: "Pmax", X: 0.1, Y: 0.1, Width: 0.45, Height: 0.2, Mode: "handwriting"}},
	}
	d, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	regions := d.Pages[0].Images[0].TextRegions
	if len(regions) != 1 || regions[0].Text != "Pmax" || regions[0].Mode != "handwriting" {
		t.Fatalf("manual OCR region was not preserved: %+v", regions)
	}
	asset.TextRegions[0].Text = ""
	asset.TextRegions[0].Mode = "erase"
	d, err = f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil || len(d.Pages[0].Images[0].TextRegions) != 1 {
		t.Fatalf("erase override was not accepted: err=%v", err)
	}
}

func TestMovedOCRRegionKeepsOriginalEraseBounds(t *testing.T) {
	f := testFont(t)
	asset := ImageAsset{
		ID: "ocr-moved", Page: 0, Data: testImageData(t), X: 20, Y: 20, Width: 50, Height: 50,
		TextRegions: []ImageTextRegion{{ID: "label-1", Text: "P", X: .55, Y: .6, Width: .2, Height: .2, Mode: "handwriting", Source: "ocr", SourceBounds: &RegionBounds{X: .1, Y: .1, Width: .2, Height: .2}}},
	}
	d, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	r := d.Pages[0].Images[0].TextRegions[0]
	if r.X != .55 || r.SourceBounds == nil || r.SourceBounds.X != .1 {
		t.Fatalf("moved region lost source bounds: %+v", r)
	}
}
