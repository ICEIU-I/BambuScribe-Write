package engine

import (
	"regexp"
	"strings"
	"unicode"
)

var diagramLabelCommand = regexp.MustCompile(`\\[a-zA-Z]+`)
var diagramPrimeScript = regexp.MustCompile(`\^\{\s*[′″](?:\s*[′″])*\s*\}|\^[′″]`)

// Scientific labels use compact CJK text. Keep separators between English
// words and values/units, but remove model-inserted padding around Chinese.
func compactDiagramLabel(text string) string {
	runes := []rune(strings.TrimSpace(text))
	var out strings.Builder
	for i := 0; i < len(runes); {
		if !unicode.Is(unicode.Zs, runes[i]) {
			out.WriteRune(runes[i])
			i++
			continue
		}
		end := i + 1
		for end < len(runes) && unicode.Is(unicode.Zs, runes[end]) {
			end++
		}
		if i > 0 && end < len(runes) && !unicode.Is(unicode.Han, runes[i-1]) && !unicode.Is(unicode.Han, runes[end]) {
			out.WriteByte(' ')
		}
		i = end
	}
	return out.String()
}

// Prime glyphs already sit above the baseline. Normalize the model's LaTeX
// spelling before selecting the text parser, without raising/shrinking them
// again or changing the original scene. Other commands stay unsupported.
func normalizeDiagramPrimes(text string) string {
	text = diagramLabelCommand.ReplaceAllStringFunc(text, func(command string) string {
		if command == `\prime` {
			return "′"
		}
		return command
	})
	var out strings.Builder
	end := 0
	for _, match := range diagramPrimeScript.FindAllStringIndex(text, -1) {
		base := strings.TrimRightFunc(text[:match[0]], unicode.IsSpace)
		if base == "" || strings.ContainsRune(`{_^\`, rune(base[len(base)-1])) {
			continue
		}
		out.WriteString(text[end:match[0]])
		for _, r := range text[match[0]:match[1]] {
			if r == '′' || r == '″' {
				out.WriteRune(r)
			}
		}
		end = match[1]
	}
	out.WriteString(text[end:])
	return strings.ReplaceAll(out.String(), "′′", "″")
}
