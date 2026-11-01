package tableimport

import (
	"fmt"
	"strings"
	"unicode"
)

// Keep subscripts, mean bars, radicals, values and writable blanks attached.
// Chinese prose can wrap at character boundaries.
func wrapTokens(text string) []string {
	runes := []rune(text)
	tokens := []string{}
	for i := 0; i < len(runes); {
		start := i
		r := runes[i]
		i++
		if r == '√' && i < len(runes) && strings.ContainsRune("(（{", runes[i]) {
			i = endWrapGroup(runes, i)
		} else if r == '_' && i < len(runes) && runes[i] == '_' {
			for i < len(runes) && runes[i] == '_' {
				i++
			}
		} else if unicode.IsLetter(r) && !unicode.Is(unicode.Han, r) || unicode.IsDigit(r) {
			for i < len(runes) && ((unicode.IsLetter(runes[i]) && !unicode.Is(unicode.Han, runes[i])) || unicode.IsDigit(runes[i]) || strings.ContainsRune(".⁺⁻⁰¹²³⁴⁵⁶⁷⁸⁹", runes[i])) {
				i++
			}
		} else if unicode.Is(unicode.Zs, r) {
			for i < len(runes) && unicode.Is(unicode.Zs, runes[i]) {
				i++
			}
		}
		for i < len(runes) {
			if unicode.Is(unicode.Mn, runes[i]) {
				i++
			} else if runes[i] == '_' && i+1 < len(runes) && runes[i+1] == '{' {
				i = endWrapGroup(runes, i+1)
			} else {
				break
			}
		}
		token := string(runes[start:i])
		if len(tokens) > 0 && (unicode.Is(unicode.Mn, r) || strings.ContainsRune(",，。；;、）)]}", r)) {
			tokens[len(tokens)-1] += token
		} else {
			tokens = append(tokens, token)
		}
	}
	// A blank answer followed by its unit must stay on one line.
	for i := len(tokens) - 2; i >= 0; i-- {
		if strings.HasSuffix(tokens[i], "=") || strings.HasSuffix(tokens[i], "＝") || strings.HasSuffix(tokens[i], ":") || strings.HasSuffix(tokens[i], "：") {
			if strings.TrimSpace(tokens[i+1]) == "" && len([]rune(tokens[i+1])) >= 2 {
				end := i + 2
				if end < len(tokens) && tokens[end] != "\n" {
					end++
				}
				tokens[i] = strings.Join(tokens[i:end], "")
				tokens = append(tokens[:i+1], tokens[end:]...)
			}
		}
	}
	return tokens
}

func endWrapGroup(runes []rune, start int) int {
	depth := 0
	for i := start; i < len(runes); i++ {
		if strings.ContainsRune("(（{", runes[i]) {
			depth++
		} else if strings.ContainsRune(")）}", runes[i]) {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(runes)
}

func wrapText(text string, width float64, measure func(string) float64) (string, error) {
	lines := []string{}
	for _, paragraph := range strings.Split(text, "\n") {
		line := ""
		for _, token := range wrapTokens(paragraph) {
			if measure(line+token) <= width+1e-8 {
				line += token
				continue
			}
			if line != "" {
				lines = append(lines, strings.TrimRight(line, " "))
				line = ""
			}
			if strings.TrimSpace(token) == "" {
				continue
			}
			if measure(token) > width+1e-8 {
				return "", fmt.Errorf("正文70%%字号下公式或填写区仍超宽，请加宽表格或纸张")
			}
			line = token
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}
