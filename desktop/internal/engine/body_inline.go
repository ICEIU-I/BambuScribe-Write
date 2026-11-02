package engine

import (
	"maps"
	"strings"
	"unicode"
)

// Each formatted span is a single wrapping/selection cell. The original source
// remains in page text, UTF-16 carets and saved projects.
type bodyScript struct {
	source   string
	base     rune
	advance  float64
	length   int
	math     bool
	fraction *imageMathBox
}
type bodySyntaxIssue struct {
	start, end int
	message    string
}

func (f *Font) bodySource(r rune) string {
	if g, ok := f.bodyScripts[r]; ok {
		return g.source
	}
	return string(r)
}
func (f *Font) bodyRuneWidth(r rune) int {
	if g, ok := f.bodyScripts[r]; ok {
		return g.length
	}
	return utf16Width(r)
}
func (f *Font) bodyBase(r rune) rune {
	if g, ok := f.bodyScripts[r]; ok {
		return g.base
	}
	return r
}
func (f *Font) bodyStrokes(r rune, g Glyph, v handwritingVariation, index int) []Stroke {
	if f.bodyScripts[r].math {
		return g.Strokes
	}
	return v.strokes(g.Strokes, index)
}
func bodyLength(runes []rune) int {
	n := 0
	for _, r := range runes {
		n += utf16Width(r)
	}
	return n
}
func bodyMathCandidate(runes []rune, i int) bool {
	r := runes[i]
	if r == '\\' {
		return true
	}
	if (r == '_' || r == '^') && i+1 < len(runes) && runes[i+1] == '{' {
		return true
	}
	if r == '√' && i+1 < len(runes) && strings.ContainsRune("(（{", runes[i+1]) {
		return true
	}
	if !unicode.IsSpace(r) && !strings.ContainsRune("_{}^", r) && i+2 < len(runes) && strings.ContainsRune("_^", runes[i+1]) && runes[i+2] == '{' {
		return true
	}
	if r == '(' || r == '（' || r == '{' {
		end := bodyGroupEnd(runes, i)
		return end < len(runes) && strings.ContainsRune("_^", runes[end])
	}
	return false
}

// Find the end of an editing candidate even when parsing failed inside it.
// Never reinterpret a nested command within a broken outer structure.
func bodyGroupEnd(runes []rune, start int) int {
	pairs := map[rune]rune{'{': '}', '(': ')', '（': '）', '[': ']'}
	stack := []rune{pairs[runes[start]]}
	for i := start + 1; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) && strings.ContainsRune(`_^{}\`, runes[i+1]) {
			i++
			continue
		}
		if close, ok := pairs[runes[i]]; ok {
			stack = append(stack, close)
		} else if strings.ContainsRune("})）]", runes[i]) {
			if runes[i] != stack[len(stack)-1] {
				return i + 1
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i + 1
			}
		}
	}
	return len(runes)
}
func bodySyntaxEnd(runes []rune, start int) int {
	i := start + 1
	if runes[start] == '\\' {
		for i < len(runes) && unicode.IsLetter(runes[i]) {
			i++
		}
	} else if strings.ContainsRune("(（{", runes[start]) {
		i = bodyGroupEnd(runes, start)
	}
	for i < len(runes) {
		if strings.ContainsRune("{(（[", runes[i]) {
			i = bodyGroupEnd(runes, i)
			continue
		}
		if strings.ContainsRune("_^", runes[i]) && i+1 < len(runes) && runes[i+1] == '{' {
			i = bodyGroupEnd(runes, i+1)
			continue
		}
		break
	}
	return i
}

func (f *Font) bodyRunes(text string, s LayoutSettings, variations ...handwritingVariation) (*Font, []rune) {
	runes := []rune(text)
	if !strings.ContainsAny(text, "_^{\\√") {
		return f, runes
	}
	var variation handwritingVariation
	if len(variations) > 0 {
		variation = variations[0]
	}
	font := f
	clone := func() {
		if font == f {
			copy := *f
			copy.Glyphs = maps.Clone(f.Glyphs)
			copy.bodyScripts = make(map[rune]bodyScript)
			copy.bodyIssues = nil
			font = &copy
		}
	}
	out := make([]rune, 0, len(runes))
	token := rune(0xf0000)
	index := 0
	fractionRuns := bodyFractionRuns(runes)
	for i := 0; i < len(runes); {
		runEnd := fractionRuns[i]
		if runEnd == 0 && !bodyMathCandidate(runes, i) {
			out = append(out, runes[i])
			index += utf16Width(runes[i])
			i++
			continue
		}
		ratio := s.FractionScale
		if ratio == 0 {
			ratio = 85
		}
		parser := formulaParser{font: f, input: runes[i:], inline: true, settings: s, ratio: ratio / 100, variation: variation.at(index)}
		b, err := parser.atom(0)
		if runEnd > 0 {
			grouped := formulaParser{font: f, input: runes[i:runEnd], inline: true, settings: bodyFractionSettings(s), ratio: ratio / 100, variation: variation.at(index)}
			if box, problem := grouped.sequence(0, 0); problem == nil && grouped.hasFraction {
				b, err, parser = box, nil, grouped
			}
		}
		if err != nil {
			end := bodySyntaxEnd(runes, i)
			clone()
			length := bodyLength(runes[i:end])
			font.bodyIssues = append(font.bodyIssues, bodySyntaxIssue{index, index + length, err.Error()})
			out = append(out, runes[i:end]...)
			index += length
			i = end
			continue
		}
		end := i + parser.pos
		length := bodyLength(runes[i:end])
		source := string(runes[i:end])
		g := Glyph{Width: b.width, Strokes: appendMathStrokes(nil, b.strokes, 0, f.Units, 1)}
		advance := (b.width + parser.gap(0)) / f.Units * s.FontSize
		fixed := true
		// Keep existing simple subscripts byte-for-byte stable, including variation.
		if subEnd := bodySubscriptEnd(runes, i); subEnd > 0 && subEnd+1 == end && f.bodyScriptSupported(runes[i], runes[i+3:subEnd]) {
			g, advance = f.bodyScriptGlyph(runes[i], runes[i+3:subEnd], s)
			fixed = false
		}
		clone()
		for {
			_, exists := font.Glyphs[token]
			if !exists && !strings.ContainsRune(text, token) {
				break
			}
			token++
		}
		font.Glyphs[token] = g
		script := bodyScript{source: source, base: runes[i], advance: advance, length: length, math: fixed}
		if parser.hasFraction {
			script.fraction = &b
		}
		font.bodyScripts[token] = script
		out = append(out, token)
		token++
		index += length
		i = end
	}
	if font != f {
		_, pitch := s.normalized().rows(0)
		font.fitBodyFractions(s, pitch)
	}
	return font, out
}
