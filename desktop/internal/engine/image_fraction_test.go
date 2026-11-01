package engine

import (
	"math"
	"testing"
)

func TestImageFractionLayoutKeepsStructureAndRotation(t *testing.T) {
	glyph := Glyph{Width: 8, Strokes: []Stroke{{{0, 0}, {6, 0}, {6, 10}, {0, 10}}}}
	f := &Font{Units: 10, Glyphs: map[rune]Glyph{}}
	for _, r := range "UHRISBd" {
		f.Glyphs[r] = glyph
	}
	f.Glyphs['='] = Glyph{Width: 8, Strokes: []Stroke{{{0, 4}, {6, 4}}, {{0, 6}, {6, 6}}}}
	asset := ImageAsset{X: 5, Y: 7, Width: 100, Height: 60}
	box := OCRBox{Text: `U_{H}=R_{H}\frac{I_{S}B}{d}`, X: .1, Y: .1, W: .8, H: .8, TextFormat: "math"}
	strokes := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 148, 76)
	if len(strokes) != 11 {
		t.Fatalf("missing formula glyph or bar: %d", len(strokes))
	}
	// U,H,= (two strokes),R,H,I,S,B,d, fraction bar.
	bar := strokes[10]
	axis := (strokes[2][0].Y + strokes[3][0].Y) / 2
	if math.Abs(bar[0].Y-axis) > 1e-9 || bar[0].Y != bar[1].Y {
		t.Fatal("fraction bar is not aligned with equals", bar, axis)
	}
	for _, i := range []int{6, 7, 8} {
		for _, p := range strokes[i] {
			if p.Y >= axis {
				t.Fatal("numerator touches/crosses bar")
			}
		}
	}
	for _, p := range strokes[9] {
		if p.Y <= axis {
			t.Fatal("denominator crosses bar")
		}
	}
	for _, pair := range [][2]int{{0, 1}, {4, 5}, {6, 7}} {
		base, sub := strokes[pair[0]], strokes[pair[1]]
		if math.Abs((sub[2].Y-sub[0].Y)/(base[2].Y-base[0].Y)-.65) > 1e-9 || sub[2].Y <= base[2].Y || sub[0].X <= base[1].X {
			t.Fatal("subscript size/position lost")
		}
	}
	for _, s := range strokes {
		for _, p := range s {
			if p.X < asset.X+box.X*asset.Width-1e-9 || p.X > asset.X+(box.X+box.W)*asset.Width+1e-9 || p.Y < asset.Y+box.Y*asset.Height-1e-9 || p.Y > asset.Y+(box.Y+box.H)*asset.Height+1e-9 {
				t.Fatal("formula exceeds source bounds")
			}
		}
	}
	asset.Rotation = 27
	rotated := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 148, 76)
	for i, s := range strokes {
		for j, p := range s {
			if distance(imagePoint(asset, p.X, p.Y), rotated[i][j]) > 1e-9 {
				t.Fatal("fraction rotation changed layout")
			}
		}
	}
	for _, bad := range []string{`\frac{}{d}`, `\frac{I}{}`, `\frac{I}{d`, `\frac{I}{d}}`, `\frac{\sqrt{I}}{d}`, `\frac{I_{S}^{B}}{d}`, `\frac{Z}{d}`} {
		box.Text = bad
		if got := imageRegionTextStrokes(f, box, asset, DefaultLayout(), 148, 76); len(got) != 0 {
			t.Fatal("invalid formula erased source", bad)
		}
	}
}

func TestFractionPrefixUsesExistingScriptPlacement(t *testing.T) {
	f := testFont(t)
	for _, text := range []string{"U_{H}", "U^{H}"} {
		p := imageMathParser{font: f, text: []rune(text)}
		box, ok := p.sequence(0, false)
		if !ok {
			t.Fatal("prefix parse failed")
		}
		asset := ImageAsset{Width: 100, Height: 100}
		// Constrain only height so differing horizontal glyph advances cannot
		// change the scale while checking the shared baseline rule.
		legacyBox := OCRBox{Text: text, X: 0, Y: 0, W: 1, H: (box.bottom - box.top) / 100, TextFormat: "math"}
		old := imageRegionTextStrokes(f, legacyBox, asset, DefaultLayout(), 100, 100)
		if len(old) != len(box.strokes) {
			t.Fatal("glyph strokes changed")
		}
		for i, s := range box.strokes {
			for j, v := range s {
				if math.Abs(old[i][j].Y-(v.Y-box.top)) > 1e-9 {
					t.Fatal("fraction script baseline differs from existing image math", text)
				}
			}
		}
	}
}
