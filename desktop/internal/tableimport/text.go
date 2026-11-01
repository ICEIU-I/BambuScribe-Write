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
		if writingBlank(runes, i, end) {
			out.WriteString(strings.Repeat(" ", end-i))
		} else if i > 0 && end < len(runes) && !compactSpaceBoundary(runes[i-1]) && !compactSpaceBoundary(runes[end]) {
			out.WriteByte(' ')
		}
		i = end
	}
	return out.String()
}

// Several spaces after an assignment (or inside empty parentheses) reserve
// room for a handwritten answer. A populated numeric value is still compact.
func writingBlank(runes []rune, start, end int) bool {
	if start == 0 || end-start < 2 {
		return false
	}
	previous := runes[start-1]
	if strings.ContainsRune("(（[【", previous) {
		return end < len(runes) && strings.ContainsRune(")）]】", runes[end])
	}
	if !strings.ContainsRune("=＝:：", previous) {
		return false
	}
	if end == len(runes) || strings.ContainsRune("\r\n,，;；)）]】", runes[end]) {
		return true
	}
	suffix := string(runes[end:])
	suffix = strings.TrimLeft(suffix, "(（")
	for _, unit := range []string{"mm", "cm", "nm", "μm", "µm", "km", "m", "ms", "s", "mV", "V", "mA", "μA", "µA", "A", "kΩ", "Ω", "Hz", "kHz", "MHz", "kg", "mg", "g", "N", "Pa", "J", "W", "°C", "℃", "%", "毫米", "厘米", "米", "秒", "伏", "安", "度"} {
		if !strings.HasPrefix(suffix, unit) {
			continue
		}
		after := []rune(strings.TrimPrefix(suffix, unit))
		if len(after) == 0 || !unicode.IsLetter(after[0]) {
			return true
		}
	}
	return false
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
