package engine

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

type tableLetter struct {
	glyph          Glyph
	char           rune
	width, bearing float64
	size, shift    float64
	missing        bool
	overline       bool
}
type tableText struct {
	variation                  handwritingVariation
	lines                      [][]tableLetter
	widths                     []float64
	lineHeight, inkTop, height float64
	issue                      string
}

// Only explicit newlines create new rows of text. Keep the font's proportions
// and baseline, and report insufficient width instead of silently wrapping.
func (f *Font) tableText(text string, size, width float64) tableText {
	return f.tableTextWithSpacing(text, size, width, nil)
}

func (f *Font) tableTextWithSpacing(text string, size, width float64, spacing *LayoutSettings) tableText {
	out := tableText{lines: [][]tableLetter{}, inkTop: math.Inf(1)}
	bottom := math.Inf(-1)
	if text == "" {
		out.inkTop = 0
		return out
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	for _, paragraph := range strings.Split(text, "\n") {
		// Trailing spaces have no ink and must not affect alignment or height.
		// Retain the original cell text, indentation and explicit blank lines.
		paragraph = strings.TrimRightFunc(paragraph, unicode.IsSpace)
		line := []tableLetter{}
		used := 0.0
		for _, character := range tableCharacters(paragraph) {
			r := character.char
			letterSize, shift := size*character.scale, size*character.shift
			g, ok := f.Glyphs[r]
			overline := false
			if ok && character.overline {
				count := len(g.Strokes)
				g = f.tableOverline(g)
				overline = len(g.Strokes) > count
			}
			l := tableLetter{overline: overline, glyph: g, char: r, size: letterSize, shift: shift, missing: !ok && !unicode.IsSpace(r)}
			if unicode.IsSpace(r) {
				l.width = letterSize * .5
				if r == '\t' {
					l.width *= 2
				}
			} else if !ok {
				l.width = letterSize
				out.inkTop = min(out.inkTop, shift)
				bottom = max(bottom, shift+letterSize)
			} else {
				left, right := math.Inf(1), math.Inf(-1)
				for _, stroke := range g.Strokes {
					for _, p := range stroke {
						left = min(left, p.X/f.Units*letterSize)
						right = max(right, p.X/f.Units*letterSize)
						out.inkTop = min(out.inkTop, shift+p.Y/f.Units*letterSize)
						bottom = max(bottom, shift+p.Y/f.Units*letterSize)
					}
				}
				if !finite(left) {
					left, right = 0, g.Width/f.Units*letterSize
				}
				l.width = max(g.Width/f.Units*letterSize, right-left) + letterSize*.12
				if fullCell(r) {
					l.width = max(l.width, letterSize)
				}
				l.bearing = (l.width-(right-left))/2 - left
			}
			if spacing != nil {
				s := *spacing
				s.FontSize = letterSize
				l.width = f.advance(r, s)
				if ok && !unicode.IsSpace(r) {
					l.bearing = textGlyphBearing(g, f.Units, l.width, s)
				}
			}
			line = append(line, l)
			used += l.width
		}
		if spacing != nil && len(line) > 0 {
			// Extra spacing separates characters, not the final character and frame.
			used -= spacing.LetterGap
		}
		if used > width+1e-8 {
			out.issue = "列宽不足，请增大列宽、减小字号或按 Shift+Enter 换行"
		}
		out.lines = append(out.lines, line)
		out.widths = append(out.widths, used)
	}
	// Fit and center the visible ink, not the font's empty em box. In the
	// handwriting font a digit can occupy less than half of that box.
	// Whitespace-only text still reserves explicit lines for editing.
	if !finite(out.inkTop) {
		out.inkTop, bottom = 0, size
	}
	out.lineHeight = max(size*1.25, bottom-out.inkTop+size*.2)
	out.height = float64(len(out.lines)-1)*out.lineHeight + bottom - out.inkTop
	return out
}

func (f *Font) placeTableText(t tableText, padding, x, y, width, height float64, align, vertical string) ([]Stroke, []MissingGlyph) {
	out := []Stroke{}
	missing := []MissingGlyph{}
	originY := y + padding - t.inkTop
	spareY := height - 2*padding - t.height
	if vertical == "middle" {
		originY += max(0, spareY) / 2
	} else if vertical == "bottom" {
		originY += max(0, spareY)
	}
	occurrence := 0
	for i, line := range t.lines {
		originX := x + padding
		spareX := width - 2*padding - t.widths[i]
		if align == "center" {
			originX += max(0, spareX) / 2
		} else if align == "right" {
			originX += max(0, spareX)
		}
		for index := 0; index < len(line); {
			if stroke, count, advance := f.tableFillLine(line[index:], originX, originY+float64(i)*t.lineHeight); count > 0 {
				out = append(out, stroke)
				originX += advance
				index += count
				continue
			}
			l := line[index]
			index++
			letterY := originY + float64(i)*t.lineHeight + l.shift
			if l.missing {
				missing = append(missing, MissingGlyph{Char: string(l.char), X: originX, Y: letterY, Size: l.size})
			}
			strokes := l.glyph.Strokes
			if l.overline {
				strokes = strokes[:len(strokes)-1]
			}
			strokes = t.variation.strokes(strokes, occurrence+index-1)
			if l.overline {
				strokes = append(append([]Stroke(nil), strokes...), l.glyph.Strokes[len(l.glyph.Strokes)-1])
			}
			for _, stroke := range strokes {
				s := make(Stroke, len(stroke))
				for j, p := range stroke {
					s[j] = Point{originX + l.bearing + p.X/f.Units*l.size, letterY + p.Y/f.Units*l.size}
				}
				if len(s) >= 2 {
					out = append(out, s)
				}
			}
			originX += l.width
		}
		occurrence += len(line) + 1
	}
	return out, missing
}

func tableCellIssue(c TableCell, issue string) string {
	return fmt.Sprintf("第 %d 行第 %d 列：%s", c.Row+1, c.Col+1, issue)
}
