package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Form feeds mark explicit sheet boundaries. Each section still uses the same
// editable text flow, object avoidance and export geometry as ordinary typing.
func (f *Font) layoutText(text string, s LayoutSettings, occupied map[int][]RegionBounds) (*Document, error) {
	if !strings.ContainsRune(text, '\f') {
		return f.layoutFlow(text, s, occupied, 0, len(s.PageLayouts) > 0 && s.PageLayouts[0].Continuation, false, 0)
	}
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxDocumentCharacters {
		return nil, fmt.Errorf("文字最多 20,000 字，请拆分文档")
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	sections := strings.Split(text, "\f")
	var doc *Document
	offset := 0
	for i, section := range sections {
		pageOffset := 0
		if doc != nil {
			pageOffset = len(doc.Pages)
		}
		local := map[int][]RegionBounds{}
		for page, bounds := range occupied {
			if page >= pageOffset {
				local[page-pageOffset] = bounds
			}
		}
		continued := pageOffset < len(s.PageLayouts) && s.PageLayouts[pageOffset].Continuation
		part, err := f.layoutFlow(section, s, local, pageOffset, continued, i < len(sections)-1, offset)
		if err != nil {
			return nil, err
		}
		for pi := range part.Pages {
			for li := range part.Pages[pi].Ink {
				n, _ := strconv.Atoi(part.Pages[pi].Ink[li].Part)
				part.Pages[pi].Ink[li].Part = strconv.Itoa(n + offset)
			}
			for fi := range part.Pages[pi].Formulas {
				part.Pages[pi].Formulas[fi].Start += offset
				part.Pages[pi].Formulas[fi].End += offset
			}
			for ci := range part.Pages[pi].Carets {
				part.Pages[pi].Carets[ci].Index += offset
			}
			for ci := range part.Pages[pi].BlankCarets {
				part.Pages[pi].BlankCarets[ci].Index += offset
			}
			for ci := range part.Pages[pi].Cells {
				part.Pages[pi].Cells[ci].Start += offset
				part.Pages[pi].Cells[ci].End += offset
			}
		}
		if doc == nil {
			doc = part
		} else {
			doc.Pages = append(doc.Pages, part.Pages...)
			for _, missing := range part.Missing {
				found := false
				for _, existing := range doc.Missing {
					if existing == missing {
						found = true
						break
					}
				}
				if !found {
					doc.Missing = append(doc.Missing, missing)
				}
			}
		}
		for _, r := range section {
			offset += utf16Width(r)
		}
		if i < len(sections)-1 {
			doc.Pages[len(doc.Pages)-1].Text += "\f"
			offset++
		}
	}
	doc.CharacterCount = utf8.RuneCountInString(strings.ReplaceAll(text, "\f", ""))
	encoded, _ := json.Marshal(struct {
		Text     string
		Settings LayoutSettings
		Occupied map[int][]RegionBounds
		Version  string
		Font     string
	}{text, doc.Settings, occupied, "explicit-pages-v9-ink-overlap", f.ID})
	h := sha256.Sum256(encoded)
	doc.ID = hex.EncodeToString(h[:12])
	return doc, nil
}
