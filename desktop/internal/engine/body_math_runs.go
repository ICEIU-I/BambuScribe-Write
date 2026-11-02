package engine

import "strings"

// Prose and spaces separate inline equations; grouped text remains part of the
// equation. Only runs containing a fraction need a shared row-height transform.
func bodyFractionRuns(runes []rune) map[int]int {
	runs := map[int]int{}
	for i := 0; i < len(runes); {
		if !bodyMathRunRune(runes[i]) {
			i++
			continue
		}
		start, depth := i, 0
		for i < len(runes) {
			r := runes[i]
			if depth == 0 && !bodyMathRunRune(r) {
				break
			}
			if r == '\\' && i+1 < len(runes) && strings.ContainsRune(`_^{}\`, runes[i+1]) {
				i += 2
				continue
			}
			if strings.ContainsRune("{[(（", r) {
				depth++
			}
			if strings.ContainsRune("}])）", r) {
				depth--
				if depth < 0 {
					break
				}
			}
			i++
		}
		if strings.Contains(string(runes[start:i]), `\frac{`) {
			runs[start] = i
		}
		if i == start {
			i++
		}
	}
	return runs
}

func bodyMathRunRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
		r >= 0x370 && r <= 0x3ff || strings.ContainsRune(`+-−±*/×÷=≈≠<>≤≥^_\{}[]()（）√|·.`, r)
}
