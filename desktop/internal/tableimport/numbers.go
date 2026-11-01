package tableimport

import (
	"regexp"
	"strings"
)

// Models sometimes format 1000 as 1 000 (including typographic spaces).
// Join only complete thousands groups; retain separate values, units, tabs,
// line breaks, decimal points, signs and sub/superscript notation.
var spacedNumber = regexp.MustCompile(`[0-9]+(?:[ \x{00a0}\x{2009}\x{202f}]+[0-9]+)+`)
var numberDigits = regexp.MustCompile(`[0-9]+`)

func normalizeNumbers(text string) string {
	return spacedNumber.ReplaceAllStringFunc(text, func(number string) string {
		groups := numberDigits.FindAllStringIndex(number, -1)
		var out strings.Builder
		out.WriteString(number[:groups[0][1]])
		canGroup := groups[0][1] <= 3
		for i, group := range groups[1:] {
			length := group[1] - group[0]
			if !canGroup || length != 3 {
				out.WriteString(number[groups[i][1]:group[0]])
				canGroup = length <= 3
			}
			out.WriteString(number[group[0]:group[1]])
		}
		return out.String()
	})
}
