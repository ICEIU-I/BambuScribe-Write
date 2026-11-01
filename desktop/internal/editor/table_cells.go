package editor

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Cell coordinates are zero-based, with inclusive range ends.
type Selection struct {
	R0 int `json:"r0"`
	C0 int `json:"c0"`
	R1 int `json:"r1"`
	C1 int `json:"c1"`
}

func selected(t *engine.TableAsset, s Selection) (Selection, []int, error) {
	if s.R0 < 0 || s.C0 < 0 || s.R1 < s.R0 || s.C1 < s.C0 || s.R1 >= len(t.RowHeights) || s.C1 >= len(t.ColumnWidths) {
		return s, nil, fmt.Errorf("单元格范围无效，行列从 0 开始，结束位置包含在范围内")
	}
	for {
		old := s
		for _, c := range t.Cells {
			if c.Row <= s.R1 && c.Row+c.RowSpan > s.R0 && c.Col <= s.C1 && c.Col+c.ColSpan > s.C0 {
				s.R0 = min(s.R0, c.Row)
				s.C0 = min(s.C0, c.Col)
				s.R1 = max(s.R1, c.Row+c.RowSpan-1)
				s.C1 = max(s.C1, c.Col+c.ColSpan-1)
			}
		}
		if old == s {
			break
		}
	}
	var indices []int
	for i, c := range t.Cells {
		if c.Row >= s.R0 && c.Col >= s.C0 && c.Row <= s.R1 && c.Col <= s.C1 {
			indices = append(indices, i)
		}
	}
	sort.Slice(indices, func(i, j int) bool {
		a, b := t.Cells[indices[i]], t.Cells[indices[j]]
		return a.Row < b.Row || a.Row == b.Row && a.Col < b.Col
	})
	return s, indices, nil
}

func mergeSplit(t *engine.TableAsset, raw json.RawMessage, split bool) error {
	var a struct {
		ID        string    `json:"id"`
		Selection Selection `json:"selection"`
	}
	if err := Decode(raw, &a); err != nil {
		return err
	}
	b, indices, err := selected(t, a.Selection)
	if err != nil {
		return err
	}
	if split {
		splitIndices(t, indices)
		return nil
	}
	first := t.Cells[indices[0]]
	var text []string
	for _, i := range indices {
		if t.Cells[i].Text != "" {
			text = append(text, t.Cells[i].Text)
		}
	}
	keep := make([]engine.TableCell, 0, len(t.Cells))
	for _, c := range t.Cells {
		if c.Row >= b.R0 && c.Row <= b.R1 && c.Col >= b.C0 && c.Col <= b.C1 {
			continue
		} else {
			keep = append(keep, c)
		}
	}
	first.Row, first.Col, first.RowSpan, first.ColSpan = b.R0, b.C0, b.R1-b.R0+1, b.C1-b.C0+1
	first.Text, first.Borders = strings.Join(text, "\n"), nil
	t.Cells = append(keep, first)
	fillTable(t)
	return nil
}
func splitIndices(t *engine.TableAsset, indices []int) {
	for _, i := range indices {
		t.Cells[i].RowSpan = 1
		t.Cells[i].ColSpan = 1
		t.Cells[i].Borders = nil
	}
	fillTable(t)
}

func setCells(t *engine.TableAsset, raw json.RawMessage) error {
	var a struct {
		ID            string    `json:"id"`
		Selection     Selection `json:"selection"`
		Text          *string   `json:"text"`
		Align         *string   `json:"align"`
		VerticalAlign *string   `json:"verticalAlign"`
		Borders       string    `json:"borders"`
	}
	if err := Decode(raw, &a); err != nil {
		return err
	}
	b, indices, err := selected(t, a.Selection)
	if err != nil {
		return err
	}
	if a.Borders != "" && a.Borders != "all" && a.Borders != "outer" && a.Borders != "none" && a.Borders != "reset" {
		return fmt.Errorf("边框须为 all、outer、none 或 reset")
	}
	boolp := func(b bool) *bool { return &b }
	for _, i := range indices {
		c := &t.Cells[i]
		if a.Text != nil {
			c.Text = *a.Text
		}
		if a.Align != nil {
			c.Align = *a.Align
		}
		if a.VerticalAlign != nil {
			c.VerticalAlign = *a.VerticalAlign
		}
		if a.Borders == "reset" {
			c.Borders = nil
		} else if a.Borders != "" {
			all, outer := a.Borders == "all", a.Borders == "outer"
			c.Borders = &engine.TableBorders{Top: boolp(all || outer && c.Row == b.R0), Bottom: boolp(all || outer && c.Row+c.RowSpan-1 == b.R1), Left: boolp(all || outer && c.Col == b.C0), Right: boolp(all || outer && c.Col+c.ColSpan-1 == b.C1)}
		}
	}
	if a.Borders != "" {
		var edge *bool
		if a.Borders != "reset" {
			edge = boolp(a.Borders != "none")
		}
		for i := range t.Cells {
			c := &t.Cells[i]
			top := c.Row+c.RowSpan == b.R0 && c.Col <= b.C1 && c.Col+c.ColSpan > b.C0
			bottom := c.Row == b.R1+1 && c.Col <= b.C1 && c.Col+c.ColSpan > b.C0
			left := c.Col+c.ColSpan == b.C0 && c.Row <= b.R1 && c.Row+c.RowSpan > b.R0
			right := c.Col == b.C1+1 && c.Row <= b.R1 && c.Row+c.RowSpan > b.R0
			if top || bottom || left || right {
				if c.Borders == nil {
					c.Borders = &engine.TableBorders{}
				}
				if top {
					c.Borders.Bottom = edge
				}
				if bottom {
					c.Borders.Top = edge
				}
				if left {
					c.Borders.Right = edge
				}
				if right {
					c.Borders.Left = edge
				}
			}
		}
	}
	return nil
}
