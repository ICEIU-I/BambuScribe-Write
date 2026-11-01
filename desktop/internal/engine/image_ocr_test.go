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

const ocrScale = 4. // raster pixels per paper millimetre in these tests

// maskAsset turns an ink mask into the pasted image the pipeline reads.
func maskAsset(t *testing.T, mask []bool, width, height int) ImageAsset {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.RGBA{255, 255, 255, 255}
			if mask[y*width+x] {
				c = color.RGBA{0, 0, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return ImageAsset{
		ID:     "ocr",
		Data:   "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
		Width:  float64(width) / ocrScale,
		Height: float64(height) / ocrScale,
	}
}

// stampStrokes paints paper-millimetre strokes back into a mask, standing in
// for printed ink of a given weight.
func stampStrokes(mask []bool, width, height int, strokes []Stroke, asset ImageAsset, radius int) {
	stamp := func(px, py int) {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if px+dx >= 0 && px+dx < width && py+dy >= 0 && py+dy < height {
					mask[(py+dy)*width+px+dx] = true
				}
			}
		}
	}
	for _, s := range strokes {
		for i, p := range s {
			x := (p.X - asset.X) / asset.Width * float64(width-1)
			y := (p.Y - asset.Y) / asset.Height * float64(height-1)
			stamp(int(math.Round(x)), int(math.Round(y)))
			if i == 0 {
				continue
			}
			q := s[i-1]
			px := (q.X - asset.X) / asset.Width * float64(width-1)
			py := (q.Y - asset.Y) / asset.Height * float64(height-1)
			steps := int(math.Max(math.Abs(x-px), math.Abs(y-py))) + 1
			for k := 0; k <= steps; k++ {
				t := float64(k) / float64(steps)
				stamp(int(math.Round(px+t*(x-px))), int(math.Round(py+t*(y-py))))
			}
		}
	}
}

// drawSymbol inks a ring with a cross through it, the shape of the light-source
// symbol that the recogniser reads as a character.
func drawSymbol(mask []bool, width, height int, cx, cy, r float64) {
	put := func(x, y int) {
		if x >= 0 && x < width && y >= 0 && y < height {
			mask[y*width+x] = true
		}
	}
	for a := 0.; a < 360; a += 1 {
		rad := a * math.Pi / 180
		x, y := int(math.Round(cx+r*math.Cos(rad))), int(math.Round(cy+r*math.Sin(rad)))
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				put(x+dx, y+dy)
			}
		}
	}
	for step := -r; step <= r; step++ {
		d := math.Round(step / math.Sqrt2)
		put(int(math.Round(cx+step)), int(math.Round(cy+d)))
		put(int(math.Round(cx+step)), int(math.Round(cy-d)))
	}
}

// A label glyph is replaced, but a symbol the recogniser mistook for the
// character beside it is not: its ink has to survive into the line art.
func TestMisreadSymbolKeepsItsRasterBesideReplacedText(t *testing.T) {
	const w, h = 120, 44
	f := testFont(t)
	mask := make([]bool, w*h)

	letterBox := OCRBox{Text: "光", X: 3. / w, Y: 6. / h, W: 22. / w, H: 30. / h}
	asset := maskAsset(t, mask, w, h)
	letter := handwritingTextStrokes(f, letterBox, asset, DefaultLayout(), w, h)
	if len(letter) == 0 {
		t.Fatal("test font cannot draw the sample character")
	}
	stampStrokes(mask, w, h, letter, asset, 1)
	drawSymbol(mask, w, h, 66, h/2, 13)

	box := OCRBox{Text: "光必", X: 1. / w, Y: 4. / h, W: 100. / w, H: 36. / h}
	runs := verifiedOcrText(f, box, mask, mask, w, h, asset)
	if len(runs) != 1 || runs[0].text != "光" {
		t.Fatalf("expected only the read character to survive, got %+v", runs)
	}
	ringRight := runs[0].source.X*float64(w) + runs[0].source.Width*float64(w)
	if ringRight > 52 {
		t.Fatalf("replacement covers the symbol at column %.0f", ringRight)
	}

	// The symbol's own pixels must reach the pen, not be erased with the label.
	kept := append([]bool(nil), mask...)
	eraseImageText(kept, w, h, []OCRBox{{X: runs[0].source.X, Y: runs[0].source.Y, W: runs[0].source.Width, H: runs[0].source.Height}})
	split := int(ringRight) + 4
	label, symbol := 0, 0
	was := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !mask[y*w+x] {
				continue
			}
			if x < split {
				label++
			} else {
				was++
				if kept[y*w+x] {
					symbol++
				}
			}
		}
	}
	if label == 0 {
		t.Fatal("sample produced no label ink to erase")
	}
	if ink := countInk(kept, w, h, 0) - countInk(kept, w, h, split); ink != 0 {
		t.Fatalf("label ink survived the replacement: %d px", ink)
	}
	if symbol != was {
		t.Fatalf("symbol lost %d of %d px to the replacement", was-symbol, was)
	}
}

func countInk(mask []bool, width, height, from int) int {
	n := 0
	for y := 0; y < height; y++ {
		for x := from; x < width; x++ {
			if mask[y*width+x] {
				n++
			}
		}
	}
	return n
}

// A box whose reading matches its ink keeps the box it came with, so every
// label that already worked today is untouched.
func TestVerifiedBoxKeepsItsShape(t *testing.T) {
	const w, h = 60, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	box := OCRBox{Text: "电", X: 8. / w, Y: 6. / h, W: 22. / w, H: 30. / h}
	asset := maskAsset(t, mask, w, h)
	stampStrokes(mask, w, h, handwritingTextStrokes(f, box, asset, DefaultLayout(), w, h), asset, 1)

	runs := verifiedOcrText(f, box, mask, mask, w, h, asset)
	if len(runs) != 1 {
		t.Fatalf("expected the box back unchanged, got %+v", runs)
	}
	if runs[0].text != box.Text || runs[0].box != (OCRBox{Text: box.Text, X: box.X, Y: box.Y, W: box.W, H: box.H}) {
		t.Fatal("a verified box was rebuilt instead of passed through")
	}
	if runs[0].source.X != box.X || runs[0].source.Width != box.W {
		t.Fatal("verified box changed the ink it covers")
	}
}

// Characters the ink does not separate cannot be checked one by one, so the box
// is left to the whole-box replacement rather than guessed at.
func TestTouchingCharactersFallBackToWholeBox(t *testing.T) {
	const w, h = 60, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	for _, x := range []int{10, 11, 12, 20, 21, 22} {
		for y := 10; y < 34; y++ {
			mask[y*w+x] = true
		}
	}
	asset := maskAsset(t, mask, w, h)
	box := OCRBox{Text: "光电", X: 5. / w, Y: 6. / h, W: 30. / w, H: 30. / h}
	runs := verifiedOcrText(f, box, mask, mask, w, h, asset)
	if len(runs) != 1 || runs[0].text != "光电" {
		t.Fatalf("expected the whole box back, got %+v", runs)
	}
}

// A run of narrow characters must still be drawn at the height of the line it
// replaces, not shrunk to the width of its own ink.
func TestNarrowRunKeepsTheLinesHeight(t *testing.T) {
	const w, h = 80, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	asset := maskAsset(t, mask, w, h)
	parent := OCRBox{Text: "1 必", X: 4. / w, Y: 6. / h, W: 60. / w, H: 30. / h}
	box := runBox("1", RegionBounds{parent.X, parent.Y, 6. / w, parent.H}, parent)
	strokes := handwritingTextStrokes(f, box, asset, DefaultLayout(), w, h)
	if len(strokes) == 0 {
		t.Fatal("no glyph")
	}
	top, bottom := math.Inf(1), math.Inf(-1)
	for _, s := range strokes {
		for _, p := range s {
			top, bottom = math.Min(top, p.Y), math.Max(bottom, p.Y)
		}
	}
	if drawn := (bottom - top) / (box.H * asset.Height); drawn < .8 {
		t.Fatalf("narrow run drawn at %.0f%% of the line height", drawn*100)
	}
}

// The check must not reject the glyph it is comparing against itself.
func TestGlyphAgreementAcceptsItsOwnInk(t *testing.T) {
	const w, h = 60, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	asset := maskAsset(t, mask, w, h)
	box := OCRBox{Text: "光", X: 6. / w, Y: 6. / h, W: 24. / w, H: 30. / h}
	stampStrokes(mask, w, h, handwritingTextStrokes(f, box, asset, DefaultLayout(), w, h), asset, 1)
	cells := splitTextCells(mask, w, h, box)
	if len(cells) != 1 {
		t.Fatalf("expected one cell, got %d", len(cells))
	}
	if score := cellAgreement(f, box, cells[0], '光', mask, w, h, asset); score < .9 {
		t.Fatalf("a glyph scored %.2f against its own ink", score)
	}
}

// And it must reject a reading whose shape is not there, which is the case the
// light-source symbol produced.
func TestGlyphAgreementRejectsASymbol(t *testing.T) {
	const w, h = 60, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	drawSymbol(mask, w, h, 30, h/2, 14)
	asset := maskAsset(t, mask, w, h)
	cells := splitTextCells(mask, w, h, OCRBox{X: .02, Y: .05, W: .96, H: .9})
	if len(cells) != 1 {
		t.Fatalf("expected one cell, got %d", len(cells))
	}
	for _, r := range []rune{'必', '光', '电'} {
		if score := cellAgreement(f, OCRBox{Text: string(r), X: .02, Y: .05, W: .96, H: .9}, cells[0], r, mask, w, h, asset); score >= glyphAgreement {
			t.Fatalf("symbol accepted as %q with %.2f", r, score)
		}
	}
}

// Splitting must not produce a region the frontend or the validator rejects.
func TestSplitRegionsValidateAndRoundTrip(t *testing.T) {
	const w, h = 120, 44
	f := testFont(t)
	mask := make([]bool, w*h)
	asset := maskAsset(t, mask, w, h)
	box := OCRBox{Text: "光", X: 3. / w, Y: 6. / h, W: 22. / w, H: 30. / h}
	stampStrokes(mask, w, h, handwritingTextStrokes(f, box, asset, DefaultLayout(), w, h), asset, 1)
	drawSymbol(mask, w, h, 66, h/2, 13)

	parent := OCRBox{Text: "光必", X: 1. / w, Y: 4. / h, W: 100. / w, H: 36. / h}
	for i, run := range verifiedOcrText(f, parent, mask, mask, w, h, asset) {
		region := ImageTextRegion{
			ID: "text-" + string(rune('1'+i)), Text: run.text,
			X: run.box.X, Y: run.box.Y, Width: run.box.W, Height: run.box.H,
			Mode: "handwriting", Source: "ocr",
			SourceBounds: &RegionBounds{run.source.X, run.source.Y, run.source.Width, run.source.Height},
		}
		if err := validateImageRegion(region); err != nil {
			t.Fatalf("split region rejected: %v", err)
		}
		if strings.TrimSpace(region.Text) == "" {
			t.Fatal("split region has no text")
		}
	}
}
