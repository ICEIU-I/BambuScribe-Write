package engine

import "unicode"

// Keep only glyphs whose ink can still reach a later writing row. A low
// descender on the left must not collide with an ascender on the right just
// because their whole-line vertical envelopes overlap.
type textInk struct {
	active          []textInkGlyph
	overlaps        []RegionBounds
	envelopeOverlap bool
}

func textFragmentOrigin(part textFragment, s LayoutSettings) float64 {
	return part.left + lineOrigin(s, 0, part.right-part.left, part.used) - s.MarginLeft
}

func textGlyphBearing(g Glyph, units, advance float64, s LayoutSettings) float64 {
	return (advance - s.LetterGap - g.Width/units*s.FontSize) / 2
}

func (ink *textInk) addLine(f *Font, runes []rune, line wrappedTextLine, s LayoutSettings, y float64, variations ...handwritingVariation) bool {
	var variation handwritingVariation
	if len(variations) > 0 {
		variation = variations[0]
	}
	index := 0
	active := ink.active[:0]
	for _, b := range ink.active {
		// Glyph coordinates are nonnegative. Prune at the row origin rather
		// than this line's top, since a later line can have a higher ascender.
		if b.Y+b.Height+bodyInkWidth > y+1e-8 {
			active = append(active, b)
		}
	}
	ink.active = active
	ink.overlaps = nil
	ink.envelopeOverlap = false
	current := []textInkGlyph{}
	fits := true
	for _, part := range line.parts {
		x := textFragmentOrigin(part, s)
		for _, r := range runes[part.start:part.end] {
			advance := f.advance(r, s)
			if !unicode.IsSpace(r) {
				b := RegionBounds{X: x, Y: y, Width: s.FontSize, Height: s.FontSize}
				glyph := textInkGlyph{RegionBounds: b}
				if g, ok := f.Glyphs[r]; ok {
					left, right, top, bottom := tableGlyphBounds(g)
					if !finite(left) {
						x += advance
						index += utf16Width(r)
						continue
					}
					scale := s.FontSize / f.Units
					bearing := textGlyphBearing(g, f.Units, advance, s)
					bearing += f.letterOffset(g, s.FontSize, bearing, advance, variation, index)
					b = RegionBounds{x + bearing + left*scale, y + top*scale, (right - left) * scale, (bottom - top) * scale}
					glyph = textInkGlyph{RegionBounds: b, source: g.Strokes, variation: variation, occurrence: index,
						origin: Point{x + bearing, y}, units: f.Units, size: s.FontSize}
				}
				for i := range ink.active {
					old := &ink.active[i]
					if b.X < old.X+old.Width-1e-8 && old.X < b.X+b.Width-1e-8 &&
						b.Y < old.Y+old.Height-1e-8 && old.Y < b.Y+b.Height-1e-8 {
						ink.envelopeOverlap = true
					}
					if glyphInkTouches(&glyph, old) {
						fits = false
						ink.overlaps = append(ink.overlaps, old.RegionBounds, b)
					}
				}
				current = append(current, glyph)
			}
			x += advance
			index += utf16Width(r)
		}
	}
	ink.active = append(ink.active, current...)
	return fits
}
