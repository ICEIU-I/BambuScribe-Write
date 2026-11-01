package engine

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

type diagramLetterInk struct {
	strokes     []Stroke
	left, right float64
	gapSlots    int
}

// Use the body's advances and bearings, then fit the visible lettering into
// the model's final box. Fixed millimeter gaps do not scale with the font.
func diagramLabelTextStrokes(f *Font, box OCRBox, asset ImageAsset, settings LayoutSettings, w, h int) []Stroke {
	if strings.Contains(box.Text, `\frac`) {
		return imageRegionTextStrokes(f, box, asset, settings, w, h)
	}
	var characters []tableCharacter
	if box.TextFormat == "math" {
		var ok bool
		characters, ok = imageMathCharacters(box.Text)
		if !ok {
			return nil
		}
	} else {
		for _, r := range box.Text {
			characters = append(characters, tableCharacter{char: r, scale: 1})
		}
	}
	s := settings.normalized()
	variation := imageTextVariation(s, asset, box)
	letters := []diagramLetterInk{}
	x, slots := 0., 0
	top, bottom := math.Inf(1), math.Inf(-1)
	left, right := math.Inf(1), math.Inf(-1)
	for index, c := range characters {
		spacing := s
		spacing.FontSize, spacing.LetterGap = f.Units*c.scale, 0
		advance := f.advance(c.char, spacing)
		if !unicode.IsSpace(c.char) {
			g, ok := f.Glyphs[c.char]
			if !ok {
				return nil
			}
			bearing := textGlyphBearing(g, f.Units, advance, spacing)
			bearing += f.letterOffset(g, spacing.FontSize, bearing, advance, variation, index)
			strokes := appendMathStrokes(nil, variation.strokes(g.Strokes, index), x+bearing, c.shift*f.Units, c.scale)
			bounds, ok := imageInkBounds(strokes)
			if !ok {
				return nil
			}
			l, t, r, b := bounds.X, bounds.Y, bounds.X+bounds.Width, bounds.Y+bounds.Height
			letters = append(letters, diagramLetterInk{strokes, l, r, slots})
			left, right, top, bottom = min(left, l), max(right, r), min(top, t), max(bottom, b)
		}
		x += advance
		slots++
		if c.char == '\t' {
			slots += tabCells - 1
		}
	}
	if len(letters) == 0 || box.W <= 0 || box.H <= 0 {
		return nil
	}
	width, height := box.W*asset.Width, box.H*asset.Height
	scale := math.Inf(1)
	if right > left {
		scale = width / (right - left)
	}
	if bottom > top {
		scale = min(scale, height/(bottom-top))
	}
	if !finite(scale) || scale <= 0 {
		return nil
	}
	extent := func(scale float64) (float64, float64) {
		l, r := math.Inf(1), math.Inf(-1)
		for _, letter := range letters {
			gap := float64(letter.gapSlots) * s.LetterGap
			l, r = min(l, letter.left*scale+gap), max(r, letter.right*scale+gap)
		}
		return l, r
	}
	l, r := extent(scale)
	if r-l > width {
		zeroLeft, zeroRight := extent(0)
		if zeroRight-zeroLeft >= width {
			return nil
		}
		low, high := 0., scale
		for range 40 {
			mid := (low + high) / 2
			a, b := extent(mid)
			if b-a > width {
				high = mid
			} else {
				low = mid
			}
		}
		scale = low
		l, _ = extent(scale)
	}
	x = asset.X + box.X*asset.Width - l
	y := asset.Y + box.Y*asset.Height + (height-(bottom-top)*scale)/2 - top*scale
	var out []Stroke
	for _, letter := range letters {
		for _, stroke := range letter.strokes {
			transformed := make(Stroke, len(stroke))
			for i, p := range stroke {
				transformed[i] = imagePoint(asset, x+p.X*scale+float64(letter.gapSlots)*s.LetterGap, y+p.Y*scale)
			}
			out = append(out, transformed)
		}
	}
	return out
}

// Give the model the same character pitch used by the renderer, so it can
// reserve enough horizontal space instead of returning a box that shrinks ink.
func (f *Font) DiagramTypography(settings LayoutSettings) string {
	s := settings.normalized()
	var heights []float64
	for r, g := range f.Glyphs {
		if unicode.Is(unicode.Han, r) {
			_, _, top, bottom := tableGlyphBounds(g)
			if bottom > top {
				heights = append(heights, bottom-top)
			}
		}
	}
	average := 0.
	for _, height := range heights {
		average += height / float64(len(heights))
	}
	if average == 0 {
		average = f.Units * .65
	}
	return fmt.Sprintf("Use the body's character spacing: %.3g%% of em, with %.3g mm fixed extra gap. In this handwriting font a typical Chinese glyph has visible ink height %.3g em; its character pitch is %.3g em (at least its ink width). Estimate caption width from the character count and this pitch at your chosen ink height. Reserve enough width for the actual lettering, then use the available image width for placement. Never add spaces or tracking to fill a line.", s.LetterSpacing, s.LetterGap, average/f.Units, s.LetterSpacing/100)
}
