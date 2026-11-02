package engine

// Use the ruled row, including space below the ordinary baseline. Connected
// fractions, roots and operators share one transform and one mathematical axis.
func fitBodyFraction(b imageMathBox, units, size, pitch float64) imageMathBox {
	_, _, top, bottom := tableGlyphBounds(Glyph{Strokes: b.strokes})
	if !finite(top) || !finite(bottom) || bottom <= top {
		return b
	}
	pad := min(pitch*.1, max(bodyInkWidth, pitch*.025))
	scale := (pitch - 2*pad) / size * units / (bottom - top)
	shift := -units + pad/size*units - top*scale
	return imageMathBox{
		width:   b.width * scale,
		top:     top*scale + shift,
		bottom:  bottom*scale + shift,
		strokes: appendMathStrokes(nil, b.strokes, 0, shift, scale),
	}
}

func bodyFractionSettings(s LayoutSettings) LayoutSettings {
	s.LetterSpacing = max(s.LetterSpacing, 100)
	s.LetterGap = max(s.LetterGap, s.FontSize*.05)
	return s
}

// Refit from the original geometry when a paragraph crosses onto a page with
// different row spacing. Never scale an already fitted glyph a second time.
func (f *Font) fitBodyFractions(s LayoutSettings, pitch float64) {
	if f.bodyFractionPitch == pitch {
		return
	}
	for r, script := range f.bodyScripts {
		if script.fraction == nil {
			continue
		}
		b := fitBodyFraction(*script.fraction, f.Units, s.FontSize, pitch)
		f.Glyphs[r] = Glyph{Width: b.width, Strokes: appendMathStrokes(nil, b.strokes, 0, f.Units, 1)}
		script.advance = b.width/f.Units*s.FontSize + s.LetterGap
		f.bodyScripts[r] = script
	}
	f.bodyFractionPitch = pitch
}
