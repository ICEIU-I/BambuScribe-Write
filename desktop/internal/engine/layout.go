package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxDocumentCharacters = 20000
const lineOverlapIssue = "行距过小，笔画会重叠，请增大行距或减小字号"
const tabCells = 4

func (s LayoutSettings) Validate() error {
	s = s.normalized()
	if !finite(s.HandwritingVariation) || s.HandwritingVariation < 0 || s.HandwritingVariation > 100 {
		return fmt.Errorf("手写变化强度须为 0–100%%，请在排版设置中调整")
	}
	if !finite(s.FractionScale) || s.FractionScale != 0 && (s.FractionScale < 50 || s.FractionScale > 100) {
		return fmt.Errorf("分式字号比例须为50–100%%")
	}
	if s.FontID != "" && s.FontID != FontHandwriting && s.FontID != FontDC {
		return fmt.Errorf("未知字体 %q，请在排版设置中重新选择字体", s.FontID)
	}
	if len(s.PageLayouts) > maxDocumentCharacters {
		return fmt.Errorf("页面数量过多，请拆分文档")
	}
	for _, page := range s.PageLayouts {
		if page.LineCount < 1 || page.LineCount > 101 {
			return fmt.Errorf("页面行数无效，请重新设置行数")
		}
	}
	if s.LineMode != "fixed" && s.LineMode != "random" {
		return fmt.Errorf("请选择固定行数或随机行数")
	}
	if s.LineCount < 1 || s.LineCount > 100 {
		return fmt.Errorf("每页行数须为 1–100")
	}
	if s.Align != "" && s.Align != "left" && s.Align != "center" && s.Align != "right" {
		return fmt.Errorf("未知的对齐方式")
	}
	for _, v := range []float64{s.PaperWidth, s.PaperHeight, s.MarginTop, s.MarginRight, s.MarginBottom, s.MarginLeft, s.FontSize, s.LetterGap, s.LetterSpacing, s.LineSpacing, s.LineHeight, s.ParagraphGap, s.Indent} {
		if !finite(v) {
			return fmt.Errorf("排版参数必须为有效数字")
		}
	}
	if s.PaperWidth < 20 || s.PaperWidth > 1000 || s.PaperHeight < 20 || s.PaperHeight > 1000 {
		return fmt.Errorf("纸张尺寸须为 20–1000 mm")
	}
	if s.FontSize < 1 || s.FontSize > 40 {
		return fmt.Errorf("字框大小须为 1–40 mm")
	}
	if s.MarginTop < 0 || s.MarginRight < 0 || s.MarginBottom < 0 || s.MarginLeft < 0 || s.LetterGap < 0 || s.ParagraphGap < 0 || s.Indent < 0 {
		return fmt.Errorf("边距、字距、段距和缩进不能为负数")
	}
	if s.LetterSpacing < 50 || s.LetterSpacing > 200 || s.LineSpacing < 50 || s.LineSpacing > 200 {
		return fmt.Errorf("字距和行距须在 50–200%% 之间")
	}
	if s.LetterGap > 40 || s.ParagraphGap > 100 || s.Indent > 10 {
		return fmt.Errorf("额外字距最多 40 mm，段距最多 100 mm，缩进最多 10 字")
	}
	if s.PaperWidth-s.MarginLeft-s.MarginRight < (s.Indent+2)*(s.FontSize*s.LetterSpacing/100+s.LetterGap) || s.PaperHeight-s.MarginTop-s.MarginBottom <= 0 {
		return fmt.Errorf("正文区域太小，请减小字号、缩进或页边距")
	}
	return nil
}

func fullCell(r rune) bool {
	// CJK ideographs keep a full writing cell. Symbols with a supplied glyph
	// use that glyph's advance below, while unknown symbols still fall back to
	// a full cell so their missing-glyph marker remains predictable.
	return unicode.Is(unicode.Han, r) || r == '\u3000'
}
func noLineStart(r rune) bool {
	return strings.ContainsRune("，。！？；：、）》】〉”’!?,.;:)]}%…", r)
}
func noLineEnd(r rune) bool { return strings.ContainsRune("（《【〈“‘([{", r) }

func (f *Font) advance(r rune, s LayoutSettings) float64 {
	ratio := s.LetterSpacing / 100
	if r == '\t' {
		return tabCells * (s.FontSize*ratio + s.LetterGap)
	}
	if r == ' ' {
		return s.FontSize*0.5*ratio + s.LetterGap
	}
	g, ok := f.Glyphs[r]
	width := s.FontSize
	if ok && !fullCell(r) {
		width = (g.Width/f.Units + 0.18) * s.FontSize
	}
	width *= ratio
	if ok {
		// Keep wide glyphs inside their cell when tightening character pitch.
		width = max(width, g.Width/f.Units*s.FontSize)
	}
	return width + s.LetterGap
}

func (f *Font) Layout(text string, s LayoutSettings) (*Document, error) {
	font, err := f.ForLayout(s)
	if err != nil {
		return nil, err
	}
	f = font
	return f.layoutText(text, s, nil)
}

func (f *Font) layoutFlow(text string, s LayoutSettings, occupied map[int][]RegionBounds, pageOffset int, continuation, followedByPage bool, textOffset int) (*Document, error) {
	s = s.normalized()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > maxDocumentCharacters {
		return nil, fmt.Errorf("文字最多 20,000 字，请拆分文档")
	}
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	for _, r := range text {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return nil, fmt.Errorf("文字包含不可见控制字符")
		}
	}
	encoded, _ := json.Marshal(struct {
		Text     string
		Settings LayoutSettings
		Version  string
		Occupied map[int][]RegionBounds
		Font     string
	}{text, s, "stroke-fonts-v25-ink-overlap", occupied, f.ID})
	h := sha256.Sum256(encoded)
	name := f.Name
	if name == "" {
		name = "教案专用"
	}
	doc := &Document{ID: hex.EncodeToString(h[:12]), Settings: s, Pages: []Page{}, Missing: []string{}, CharacterCount: utf8.RuneCountInString(text), FontName: name}
	row := 0
	ink := textInk{}
	diagnosticBounds := map[RegionBounds]bool{}
	newPage := func() {
		doc.Pages = append(doc.Pages, blankLayoutPage(s, pageOffset+len(doc.Pages)))
		row = 0
		ink = textInk{}
		diagnosticBounds = map[RegionBounds]bool{}
	}
	newPage()
	y := s.MarginTop
	bottom := s.PaperHeight - s.MarginBottom
	width := s.PaperWidth - s.MarginLeft - s.MarginRight
	reserveLine := func() {
		page := &doc.Pages[len(doc.Pages)-1]
		if row >= page.LineCount || y >= bottom-1e-8 {
			newPage()
			y = s.MarginTop
		}
	}
	missingSet := map[rune]bool{}
	textIndex := 0
	paragraphs := strings.Split(text, "\n")
	if followedByPage && strings.HasSuffix(text, "\n") {
		paragraphs = paragraphs[:len(paragraphs)-1]
	}
	for pi, paragraph := range paragraphs {
		if source, ok := formulaSource(paragraph); ok {
			if err := f.appendBodyFormula(doc, s, source, occupied, &ink, &y, &row, &textIndex, newPage, s.variation(fmt.Sprintf("formula:%d", textOffset+textIndex))); err != nil {
				return nil, err
			}
			if pi < len(paragraphs)-1 || followedByPage && strings.HasSuffix(text, "\n") {
				doc.Pages[len(doc.Pages)-1].Text += "\n"
				textIndex++
				y += s.ParagraphGap
			}
			continue
		}
		runes := []rune(paragraph)
		if len(runes) == 0 {
			reserveLine()
			page := &doc.Pages[len(doc.Pages)-1]
			indent := s.Indent * (s.FontSize*s.LetterSpacing/100 + s.LetterGap)
			x := lineOrigin(s, indent, width-indent, 0)
			spans := availableTextSpans(s.MarginLeft+indent, s.MarginLeft+width, y, min(s.FontSize, page.LineHeight, bottom-y), occupied[len(doc.Pages)-1])
			if len(spans) > 0 {
				span := spans[0]
				x = span.left + lineOrigin(s, 0, span.right-span.left, 0) - s.MarginLeft
			}
			// Explicit blank lines have no ink to collide. Keep the writer's
			// spacing even when an image spans the whole writing area.
			page.Carets = append(page.Carets, CaretStop{textIndex, x, y})
			if pi < len(paragraphs)-1 || followedByPage && strings.HasSuffix(text, "\n") {
				page.Text += "\n"
				textIndex++
				y += page.LineHeight + s.ParagraphGap
				row++
			}
			continue
		}
		for offset := 0; offset < len(runes); {
			indent := 0.0
			if offset == 0 && !(pi == 0 && continuation) {
				indent = s.Indent * (s.FontSize*s.LetterSpacing/100 + s.LetterGap)
			}
			if f.advance(runes[offset], s)-s.LetterGap > width-indent+1e-8 {
				return nil, fmt.Errorf("字符无法放入正文，请减小字号或页边距")
			}
			reserveLine()
			page := &doc.Pages[len(doc.Pages)-1]
			line := f.wrapTextLine(runes, offset, indent, s, y, page.LineHeight, occupied[len(doc.Pages)-1])
			if line.inkBottom > bottom-s.MarginTop+1e-8 {
				return nil, fmt.Errorf("笔画超过正文高度，请减小字号或页边距")
			}
			if y+line.inkBottom > bottom+1e-8 {
				newPage()
				y = s.MarginTop
				continue
			}
			drawY, fits := f.placeVariedTextLine(&ink, runes, line, s, y, page.LineHeight, occupied[len(doc.Pages)-1], s.variation("body").at(textOffset+textIndex))
			if !fits && !slices.Contains(page.Issues, lineOverlapIssue) {
				// Keep text, pagination and carets current while the writer fixes
				// spacing. Page issues still block G-code generation and upload.
				page.Issues = append(page.Issues, lineOverlapIssue)
			}
			if !fits {
				appendOverlapDiagnostics(page, ink.overlaps, diagnosticBounds)
			}
			for _, part := range line.parts {
				f.appendTextFragment(page, runes[part.start:part.end], part, s, drawY, &textIndex, missingSet, doc, s.variation("body").at(textOffset))
				offset = part.end
			}
			y += page.LineHeight
			row++
		}
		if pi < len(paragraphs)-1 || followedByPage && strings.HasSuffix(text, "\n") {
			doc.Pages[len(doc.Pages)-1].Text += "\n"
			textIndex++
			y += s.ParagraphGap
		}
	}
	padding := 0
	if text != "" && !strings.HasSuffix(text, "\n") {
		padding = 1
		y += s.ParagraphGap
	}
	appendBlankCarets(&doc.Pages[len(doc.Pages)-1], s, occupied[len(doc.Pages)-1], row, textIndex, padding, y)
	for i := range doc.Pages {
		doc.Pages[i].TextStrokes = make([]Stroke, len(doc.Pages[i].Strokes))
		copy(doc.Pages[i].TextStrokes, doc.Pages[i].Strokes)
	}
	return doc, nil
}
