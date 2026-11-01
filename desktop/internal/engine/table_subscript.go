package engine

import "unicode"

type tableCharacter struct {
	char         rune
	scale, shift float64
	overline     bool
	radical      int // number of following characters under this radical
}

// A closed, nonempty _{...} group uses the original glyphs as subscripts.
// Incomplete or nested groups remain literal while the user is editing.
func tableCharacters(text string) []tableCharacter {
	runes := []rune(text)
	out := make([]tableCharacter, 0, len(runes))
	for i := 0; i < len(runes); {
		if runes[i] != '_' || i+1 >= len(runes) || runes[i+1] != '{' {
			out = appendTableCharacter(out, runes[i], 1, 0)
			i++
			continue
		}
		end, depth, nested, nonspace := i+2, 1, false, false
		for end < len(runes) {
			switch runes[end] {
			case '{':
				depth++
				nested = true
			case '}':
				depth--
			default:
				nonspace = nonspace || !unicode.IsSpace(runes[end])
			}
			if depth == 0 {
				break
			}
			end++
		}
		if depth == 0 && nonspace && !nested {
			for _, r := range runes[i+2 : end] {
				out = appendTableCharacter(out, r, .65, .4)
			}
			i = end + 1
			continue
		}
		// Consume invalid groups together so an inner marker is not interpreted.
		end = min(end+1, len(runes))
		for _, r := range runes[i:end] {
			out = append(out, tableCharacter{char: r, scale: 1})
		}
		i = end
	}
	return out
}
