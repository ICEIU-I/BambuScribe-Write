package engine

import (
	"math"
	"strings"
	"unicode"
)

// Size by actual glyph ink bounds and advances. Some handwriting glyphs extend
// beyond their nominal Width; advancing by Width alone overprinted neighbours.
func handwritingTextStrokes(f *Font, box OCRBox, asset ImageAsset, s LayoutSettings, pixelWidth, pixelHeight int) []Stroke {
	return handwritingCharacters(f, box, asset, pixelWidth, pixelHeight, nil, imageTextVariation(s, asset, box))
}

func handwritingCharacters(f *Font, box OCRBox, asset ImageAsset, pixelWidth, pixelHeight int, characters []tableCharacter, variation handwritingVariation) []Stroke {
	text := strings.TrimSpace(box.Text)
	if text == "" || pixelWidth < 1 || pixelHeight < 1 || box.W <= 0 || box.H <= 0 {
		return nil
	}
	type letter struct {
		glyph                Glyph
		left, width, advance float64
		scale, shift         float64
	}
	letters := []letter{}
	total := 0.
	top, bottom := math.Inf(1), math.Inf(-1)
	if characters == nil {
		for _, r := range text {
			characters = append(characters, tableCharacter{char: r, scale: 1})
		}
	}
	for _, character := range characters {
		r := character.char
		scale, shift := character.scale, character.shift*f.Units
		if unicode.IsSpace(r) {
			letters = append(letters, letter{advance: f.Units * .45 * scale})
			total += f.Units * .45 * scale
			continue
		}
		g, ok := f.Glyphs[r]
		if !ok {
			return nil
		}
		left, right := math.Inf(1), math.Inf(-1)
		for _, s := range g.Strokes {
			for _, p := range s {
				left = math.Min(left, p.X*scale)
				right = math.Max(right, p.X*scale)
				top = math.Min(top, p.Y*scale+shift)
				bottom = math.Max(bottom, p.Y*scale+shift)
			}
		}
		if !finite(left) {
			return nil
		}
		width := right - left
		advance := math.Max(g.Width*scale, width) + f.Units*.16*scale
		if fullCell(r) {
			advance = math.Max(advance, f.Units*scale)
		}
		letters = append(letters, letter{g, left, width, advance, scale, shift})
		total += advance
	}
	if !finite(top) {
		return nil
	}
	// Fit the ink including the last letter, not its trailing spacing.
	if len(letters) > 0 {
		last := letters[len(letters)-1]
		total -= last.advance - last.width
	}
	scale := math.Inf(1)
	if total > 0 {
		scale = box.W * asset.Width / total
	}
	if bottom > top {
		scale = math.Min(scale, box.H*asset.Height/(bottom-top))
	}
	if !finite(scale) {
		return nil
	}
	x := asset.X + box.X*asset.Width
	y := asset.Y + box.Y*asset.Height + (box.H*asset.Height-(bottom-top)*scale)/2
	out := []Stroke{}
	for index, l := range letters {
		for _, s := range variation.strokes(l.glyph.Strokes, index) {
			stroke := make(Stroke, len(s))
			for i, p := range s {
				stroke[i] = imagePoint(asset, x+(p.X*l.scale-l.left)*scale, y+(p.Y*l.scale+l.shift-top)*scale)
			}
			out = append(out, stroke)
		}
		x += l.advance * scale
	}
	return out
}

// Mask only label-sized connected components. A Vision rectangle can overlap
// an axis or curve near a subscript; erasing its entire rectangle cut the data
// line, even though that long connected component wasn't text.
func eraseImageText(mask []bool, width, height int, boxes []OCRBox) {
	extractImageText(mask, width, height, boxes)
}

// Extract whole contained components, leaving crossing diagram lines alone.
func extractImageText(mask []bool, width, height int, boxes []OCRBox) []bool {
	if len(boxes) == 0 {
		return nil
	}
	extracted := make([]bool, len(mask))
	seen := make([]bool, len(mask))
	for start, ink := range mask {
		if !ink || seen[start] {
			continue
		}
		seen[start] = true
		pixels := []int{start}
		left, right, top, bottom := start%width, start%width, start/width, start/width
		for head := 0; head < len(pixels); head++ {
			idx := pixels[head]
			x, y := idx%width, idx/width
			left = min(left, x)
			right = max(right, x)
			top = min(top, y)
			bottom = max(bottom, y)
			for _, d := range skeletonDirections {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				n := ny*width + nx
				if mask[n] && !seen[n] {
					seen[n] = true
					pixels = append(pixels, n)
				}
			}
		}
		for _, b := range boxes {
			l, r := b.X*float64(width)-2, (b.X+b.W)*float64(width)+2
			t, bt := b.Y*float64(height)-2, (b.Y+b.H)*float64(height)+2
			if float64(left) < l || float64(right) > r || float64(top) < t || float64(bottom) > bt {
				continue
			}
			for _, idx := range pixels {
				mask[idx] = false
				extracted[idx] = true
			}
			break
		}
	}
	return extracted
}
