package engine

import (
	"strings"
	"unicode"
)

// The local formula worker emits a small normalized grammar, never raw LaTeX.
// Unsupported expressions retain their original ink instead of drawing
// markup or flattening a subscript onto the main baseline.
func imageMathCharacters(text string) ([]tableCharacter, bool) {
	runes := []rune(text)
	out := []tableCharacter{}
	for i := 0; i < len(runes); {
		r := runes[i]
		if r == '_' || r == '^' {
			if i == 0 || runes[i-1] == '}' || i+2 >= len(runes) || runes[i+1] != '{' {
				return nil, false
			}
			end := i + 2
			for end < len(runes) && runes[end] != '}' {
				if !unicode.IsLetter(runes[end]) && !unicode.IsDigit(runes[end]) && runes[end] != '+' && runes[end] != '-' {
					return nil, false
				}
				end++
			}
			if end == i+2 || end == len(runes) {
				return nil, false
			}
			shift := .4
			if r == '^' {
				shift = -.4
			}
			for _, c := range runes[i+2 : end] {
				out = append(out, tableCharacter{char: c, scale: .65, shift: shift})
			}
			i = end + 1
			continue
		}
		if r == '{' || r == '}' || r == '\\' {
			return nil, false
		}
		out = append(out, tableCharacter{char: r, scale: 1})
		i++
	}
	return out, len(out) > 0
}

func imageRegionTextStrokes(f *Font, box OCRBox, asset ImageAsset, s LayoutSettings, w, h int) []Stroke {
	if box.TextFormat == "" {
		return handwritingTextStrokes(f, box, asset, s, w, h)
	}
	if box.TextFormat != "math" {
		return nil
	}
	if strings.Contains(box.Text, `\frac`) {
		return imageFractionStrokes(f, box, asset, imageTextVariation(s, asset, box))
	}
	chars, ok := imageMathCharacters(box.Text)
	if !ok {
		return nil
	}
	return handwritingCharacters(f, box, asset, w, h, chars, imageTextVariation(s, asset, box))
}
