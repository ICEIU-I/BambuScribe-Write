package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

func (f *Font) renderTable(t TableAsset, settings LayoutSettings) TablePlacement {
	t.FontSize = f.tableFontSize(t)
	rows := append([]float64(nil), t.RowHeights...)
	xs := tableOffsets(t.ColumnWidths)
	plans := make([]tableText, len(t.Cells))
	sizes := make([]float64, len(t.Cells))
	out := TablePlacement{ID: t.ID, Kind: t.Kind, FontSize: t.FontSize, Padding: t.Padding, X: t.X, Y: t.Y, ColumnWidths: t.ColumnWidths, Cells: []TableCellPlacement{}, Issues: []string{}}
	for i, c := range t.Cells {
		width := xs[c.Col+c.ColSpan] - xs[c.Col] - 2*t.Padding
		sizes[i] = f.tableCellFontSize(t, c)
		if c.Text == "" {
			sizes[i] = t.FontSize
		}
		plans[i] = f.tableText(c.Text, sizes[i], width)
		if width <= 0 {
			plans[i].issue = "内边距过大，请减小内边距或增大列宽"
		}
		if plans[i].issue != "" {
			issue := tableCellIssue(c, plans[i].issue)
			if t.isTextBox() {
				issue = "宽度不足，请加宽文本框或按 Shift+Enter 换行"
			}
			out.Issues = append(out.Issues, issue)
		}
	}
	// Non-spanning cells first, then spanning cells grow only the final row of
	// their region. A later growth cannot shrink or overlap an earlier cell.
	for _, spanning := range []bool{false, true} {
		for i, c := range t.Cells {
			if (c.RowSpan > 1) != spanning {
				continue
			}
			height := 0.0
			for _, v := range rows[c.Row : c.Row+c.RowSpan] {
				height += v
			}
			required := plans[i].height + 2*t.Padding
			if required > height+1e-8 {
				rows[c.Row+c.RowSpan-1] += required - height
			}
		}
	}
	ys := tableOffsets(rows)
	out.RowHeights = rows
	out.Width = xs[len(xs)-1]
	out.Height = ys[len(ys)-1]
	out.Strokes = tableLines(t, rows)
	for i, c := range t.Cells {
		x, y := xs[c.Col], ys[c.Row]
		w, h := xs[c.Col+c.ColSpan]-x, ys[c.Row+c.RowSpan]-y
		a, v := c.Align, c.VerticalAlign
		if a == "" {
			a = t.Align
		}
		if v == "" {
			v = t.VerticalAlign
		}
		strokes, missing := f.placeTableText(plans[i], t.Padding, x, y, w, h, a, v)
		textWidth := 0.0
		for _, width := range plans[i].widths {
			textWidth = max(textWidth, width)
		}
		out.Cells = append(out.Cells, TableCellPlacement{TableCell: c, X: x, Y: y, Width: w, Height: h, FontSize: sizes[i], TextWidth: textWidth, TextHeight: plans[i].height, Strokes: strokes, Missing: missing})
		out.Strokes = append(out.Strokes, strokes...)
	}
	if t.X+out.Width > settings.PaperWidth+1e-8 || t.Y+out.Height > settings.PaperHeight+1e-8 {
		out.Issues = append(out.Issues, fmt.Sprintf("%s超出纸张，请移动或缩小尺寸", t.label()))
	}
	return out
}

// LayoutWithObjects keeps overlapping object anchors and moves whole body
// lines clear of their bounds. Preview and export share the resulting geometry.
func (f *Font) LayoutWithObjects(text string, s LayoutSettings, images []ImageAsset, tables []TableAsset) (*Document, error) {
	font, err := f.ForLayout(s)
	if err != nil {
		return nil, err
	}
	f = font
	if err := ValidateTables(tables); err != nil {
		return nil, err
	}
	count := utf8.RuneCountInString(strings.ReplaceAll(text, "\f", ""))
	for _, t := range tables {
		for _, c := range t.Cells {
			count += utf8.RuneCountInString(c.Text)
		}
	}
	if count > maxDocumentCharacters {
		return nil, fmt.Errorf("正文和表格文字共最多 20,000 字，请拆分文档")
	}
	s = s.normalized()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	plan, err := f.planObjects(s, images, tables)
	if err != nil {
		return nil, err
	}
	if err := f.prepareBodyOccupancy(&plan, s); err != nil {
		return nil, err
	}
	doc, err := f.layoutText(text, s, plan.bodyOccupied)
	if err != nil {
		return nil, err
	}
	doc.ObjectLayout = plan.positions
	appendImages(doc, plan.images, plan.imageRenders)
	for ti, t := range plan.tables {
		for len(doc.Pages) <= t.Page {
			doc.Pages = append(doc.Pages, blankLayoutPage(doc.Settings, len(doc.Pages)))
		}
		render := plan.renders[ti]
		p := &doc.Pages[t.Page]
		p.Tables = append(p.Tables, render)
		for _, issue := range render.Issues {
			p.Issues = append(p.Issues, fmt.Sprintf("%s %d：%s", t.label(), len(p.Tables), issue))
		}
		for _, stroke := range render.Strokes {
			translated := make(Stroke, len(stroke))
			for i, point := range stroke {
				translated[i] = Point{point.X + t.X, point.Y + t.Y}
			}
			p.Strokes = append(p.Strokes, translated)
		}
		for _, c := range render.Cells {
			for _, missing := range c.Missing {
				missing.X += t.X
				missing.Y += t.Y
				p.Missing = append(p.Missing, missing)
				found := false
				for _, char := range doc.Missing {
					if char == missing.Char {
						found = true
						break
					}
				}
				if !found {
					doc.Missing = append(doc.Missing, missing.Char)
				}
			}
		}
	}
	encoded, _ := json.Marshal(struct {
		Base    string
		Tables  []TableAsset
		Images  []ImageAsset
		Version string
	}{doc.ID, tables, images, "overlapping-objects-v8-writing-order"})
	h := sha256.Sum256(encoded)
	doc.ID = hex.EncodeToString(h[:12])
	doc.CharacterCount = count
	appendObjectInk(doc, images, tables)
	for i := range doc.Pages {
		doc.Pages[i].Strokes = erasedPageInk(doc.Pages[i], nil)
	}
	return doc, nil
}
func tableOverlaps(x, y, w, h, ox, oy, ow, oh float64) bool {
	return x < ox+ow && ox < x+w && y < oy+oh && oy < y+h
}
