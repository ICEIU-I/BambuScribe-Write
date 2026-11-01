package engine

import "strings"

// Parentheses immediately after √ delimit its scope, rather than printing a
// detached root glyph next to a parenthesized expression. Incomplete input
// stays literal so editing never silently loses characters.
func tableRadicalCharacters(chars []tableCharacter, depth int) []tableCharacter {
	if depth >= 8 {
		return chars
	}
	out := make([]tableCharacter, 0, len(chars))
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		if c.char == '√' && i+1 < len(chars) {
			if end := tableRadicalGroupEnd(chars, i+1); end > i+2 {
				inside := tableRadicalCharacters(chars[i+2:end], depth+1)
				c.radical = len(inside)
				out = append(out, c)
				out = append(out, inside...)
				i = end
				continue
			}
		}
		out = append(out, c)
	}
	return out
}

func tableRadicalGroupEnd(chars []tableCharacter, start int) int {
	const open, close = "(（{", ")）}"
	closing := []rune(close)
	stack := []rune{}
	for i := start; i < len(chars); i++ {
		c := chars[i]
		if c.scale != chars[start].scale || c.shift != chars[start].shift {
			continue // Subscript punctuation cannot close a main-line group.
		}
		if j := strings.IndexRune(open, c.char); j >= 0 {
			// IndexRune returns a byte offset; these delimiters include CJK.
			index := len([]rune(open[:j]))
			stack = append(stack, closing[index])
		} else if strings.ContainsRune(close, c.char) {
			if len(stack) == 0 || c.char != stack[len(stack)-1] {
				return -1
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i
			}
		} else if i == start {
			return -1
		}
	}
	return -1
}

// Build roots from right to left so an outer bar clears nested roots, mean
// bars and subscripts. The radicand retains the font's original proportions.
func (f *Font) fitTableRadicals(line []tableLetter, chars []tableCharacter, spacing *LayoutSettings) {
	for i := len(line) - 1; i >= 0; i-- {
		count := chars[i].radical
		if count == 0 {
			continue
		}
		l := &line[i]
		w, top, bottom := 0.0, 0.0, 0.0
		hasInk := false
		for _, next := range line[i+1 : i+1+count] {
			_, _, t, b := tableGlyphBounds(next.glyph)
			if next.missing {
				t, b = 0, f.Units
			}
			if finite(t) {
				t = next.shift + t/f.Units*next.size
				b = next.shift + b/f.Units*next.size
				if !hasInk {
					top, bottom, hasInk = t, b, true
				} else {
					top, bottom = min(top, t), max(bottom, b)
				}
			}
			w += next.width
		}
		if !hasInk {
			top, bottom = l.shift, l.shift+l.size
		}
		stem, gap := l.size*.5, 0.0
		if spacing != nil {
			stem = max(stem, stem*spacing.LetterSpacing/100)
			gap = spacing.LetterGap
		}
		l.width, l.bearing, l.missing, l.radical = stem+gap, 0, false, count
		bar := top - l.size*.14
		h := bottom - top
		strokes := []Stroke{
			{{0, top + h*.6}, {stem * .22, top + h*.5}, {stem * .45, bottom}, {stem * .85, bar}},
			{{stem * .85, bar}, {l.width + w - gap, bar}},
		}
		for _, stroke := range strokes {
			for j, p := range stroke {
				stroke[j] = Point{p.X / l.size * f.Units, (p.Y - l.shift) / l.size * f.Units}
			}
		}
		l.glyph = Glyph{Width: l.width / l.size * f.Units, Strokes: strokes}
	}
}
