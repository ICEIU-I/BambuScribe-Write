package engine

// An inline fraction, including any enclosing root or attached scripts, fits
// the ordinary writing band. Scale the complete structure uniformly around
// the math axis so its bar stays level with neighboring equality signs.
// Display formulas keep their independently configured fraction size.
func fitBodyFraction(b imageMathBox, axis, units float64) imageMathBox {
	_, _, top, bottom := tableGlyphBounds(Glyph{Strokes: b.strokes})
	if !finite(top) || !finite(bottom) {
		return b
	}
	upper, lower := -units*.90, -units*.18
	scale := 1.0
	if top < upper {
		scale = min(scale, (axis-upper)/(axis-top))
	}
	if bottom > lower {
		scale = min(scale, (lower-axis)/(bottom-axis))
	}
	shift := axis * (1 - scale)
	return imageMathBox{
		width:   b.width * scale,
		top:     top*scale + shift,
		bottom:  bottom*scale + shift,
		strokes: appendMathStrokes(nil, b.strokes, 0, shift, scale),
	}
}
