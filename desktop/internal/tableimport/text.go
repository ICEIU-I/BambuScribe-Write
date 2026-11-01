package tableimport

import (
	"strings"
	"unicode"
)

// Normalize model typography before review and placement. Keep semantic
// separators (English words, units, tabs and newlines), not CJK/equals padding.
func normalizeText(text string) string {
	runes := []rune(normalizeNumbers(text))
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
		if i > 0 && end < len(runes) && !compactSpaceBoundary(runes[i-1]) && !compactSpaceBoundary(runes[end]) {
			out.WriteByte(' ')
		}
		i = end
	}
	return out.String()
}

func compactSpaceBoundary(r rune) bool {
	return unicode.Is(unicode.Han, r) || strings.ContainsRune("=＝，。；：、！？（）【】《》\r\n", r)
}

func (e *Extraction) normalizeText() {
	for i := range e.Tables {
		t := &e.Tables[i]
		t.Title = normalizeText(t.Title)
		for j := range t.Notes {
			t.Notes[j] = normalizeText(t.Notes[j])
		}
		for j := range t.Cells {
			t.Cells[j].Text = normalizeText(t.Cells[j].Text)
		}
	}
}
