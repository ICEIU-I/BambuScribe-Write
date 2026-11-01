package editor

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"sort"
)

type tableInsert struct {
	ID            string   `json:"id"`
	Page          int      `json:"page"`
	X             float64  `json:"x"`
	Y             float64  `json:"y"`
	Rows          int      `json:"rows"`
	Cols          int      `json:"cols"`
	Width         float64  `json:"width"`
	Height        float64  `json:"height"`
	Style         string   `json:"style"`
	FontSize      float64  `json:"fontSize"`
	Padding       *float64 `json:"padding"`
	Align         string   `json:"align"`
	VerticalAlign string   `json:"verticalAlign"`
	Text          string   `json:"text"`
}

func insertTable(p *project.Project, op Operation) (string, error) {
	a := tableInsert{Page: 1, Rows: 3, Cols: 3, Width: 100, Style: "grid", FontSize: p.Layout.FontSize, Align: "center", VerticalAlign: "middle"}
	if op.Tool == "textbox.insert" {
		a.Rows, a.Cols, a.Width, a.Style, a.Align = 1, 1, 35, "none", "left"
	}
	if err := Decode(op.Params, &a); err != nil {
		return "", err
	}
	if a.Page < 1 || a.Page > 20001 || a.Rows < 1 || a.Rows > 50 || a.Cols < 1 || a.Cols > 20 {
		return "", fmt.Errorf("页码须为 1–20001；表格须为 1–50 行、1–20 列")
	}
	if a.ID == "" {
		a.ID = "object-" + uuid.NewString()
	}
	if err := uniqueID(p, a.ID); err != nil {
		return "", err
	}
	padding := 1.0
	if op.Tool == "textbox.insert" {
		padding = 0.5
	}
	if a.Padding != nil {
		padding = *a.Padding
	}
	rowHeight := math.Max(8, a.FontSize*1.5+2*padding)
	if op.Tool == "textbox.insert" {
		rowHeight /= 2
	}
	if a.Height != 0 {
		rowHeight = a.Height / float64(a.Rows)
	}
	t := engine.TableAsset{ID: a.ID, Page: a.Page - 1, X: a.X, Y: a.Y, Style: a.Style, FontSize: a.FontSize, Padding: padding, Align: a.Align, VerticalAlign: a.VerticalAlign}
	if op.Tool == "textbox.insert" {
		t.Kind = "textbox"
	}
	for i := 0; i < a.Rows; i++ {
		t.RowHeights = append(t.RowHeights, rowHeight)
	}
	for i := 0; i < a.Cols; i++ {
		t.ColumnWidths = append(t.ColumnWidths, a.Width/float64(a.Cols))
	}
	fillTable(&t)
	t.Cells[0].Text = a.Text
	p.Tables = append(p.Tables, t)
	return t.ID, nil
}

func tableIndex(p *project.Project, id string) (int, error) {
	for i, t := range p.Tables {
		if t.ID == id {
			return i, nil
		}
	}
	return 0, fmt.Errorf("找不到表格或文本框 %q，请用 inspect 查看对象 ID", id)
}
func cellIndex(t *engine.TableAsset, row, col int) int {
	for i, c := range t.Cells {
		if c.Row <= row && row < c.Row+c.RowSpan && c.Col <= col && col < c.Col+c.ColSpan {
			return i
		}
	}
	return -1
}
func fillTable(t *engine.TableAsset) {
	for r := range t.RowHeights {
		for c := range t.ColumnWidths {
			if cellIndex(t, r, c) < 0 {
				t.Cells = append(t.Cells, engine.TableCell{Row: r, Col: c, RowSpan: 1, ColSpan: 1})
			}
		}
	}
	sort.Slice(t.Cells, func(i, j int) bool {
		a, b := t.Cells[i], t.Cells[j]
		return a.Row < b.Row || a.Row == b.Row && a.Col < b.Col
	})
}

// Updates merge into a typed asset, so omitted fields retain their saved values.
func editTable(p *project.Project, op Operation) (string, error) {
	var head struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(op.Params, &head); err != nil {
		return "", err
	}
	i, err := tableIndex(p, head.ID)
	if err != nil {
		return "", err
	}
	t := &p.Tables[i]
	textbox := op.Tool == "textbox.update"
	if textbox != (t.Kind == "textbox") {
		return "", fmt.Errorf("请使用与对象类型对应的表格或文本框工具")
	}
	if op.Tool == "table.update" {
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(op.Params, &fields); err != nil {
			return "", err
		}
		if raw, ok := fields["page"]; ok {
			var page int
			if err = Decode(raw, &page); err != nil {
				return "", err
			}
			if page < 1 {
				return "", fmt.Errorf("页码从 1 开始")
			}
			t.Page = page - 1
			delete(fields, "page")
		}
		body, _ := json.Marshal(fields)
		err = Decode(body, t)
		if err == nil && (t.ID != head.ID || t.Kind != "") {
			err = fmt.Errorf("不能改变对象 ID 或类型")
		}
		return head.ID, err
	}
	if textbox {
		var a struct {
			ID            string   `json:"id"`
			Text          *string  `json:"text"`
			Page          *int     `json:"page"`
			X             *float64 `json:"x"`
			Y             *float64 `json:"y"`
			Width         *float64 `json:"width"`
			Height        *float64 `json:"height"`
			FontSize      *float64 `json:"fontSize"`
			Padding       *float64 `json:"padding"`
			Align         *string  `json:"align"`
			VerticalAlign *string  `json:"verticalAlign"`
		}
		if err = Decode(op.Params, &a); err != nil {
			return "", err
		}
		if a.Text != nil {
			t.Cells[0].Text = *a.Text
		}
		if a.Page != nil {
			t.Page = *a.Page - 1
		}
		if a.X != nil {
			t.X = *a.X
		}
		if a.Y != nil {
			t.Y = *a.Y
		}
		if a.Width != nil {
			t.ColumnWidths[0] = *a.Width
		}
		if a.Height != nil {
			t.RowHeights[0] = *a.Height
		}
		if a.FontSize != nil {
			t.FontSize = *a.FontSize
		}
		if a.Padding != nil {
			t.Padding = *a.Padding
		}
		if a.Align != nil {
			t.Align = *a.Align
		}
		if a.VerticalAlign != nil {
			t.VerticalAlign = *a.VerticalAlign
		}
		return head.ID, nil
	}
	switch op.Tool {
	case "table.cells":
		err = setCells(t, op.Params)
	case "table.merge", "table.split":
		err = mergeSplit(t, op.Params, op.Tool == "table.split")
	case "table.paste":
		err = pasteTable(t, op.Params)
	default:
		var a struct {
			ID    string   `json:"id"`
			Index int      `json:"index"`
			Size  *float64 `json:"size"`
		}
		if err = Decode(op.Params, &a); err == nil {
			err = axisTable(t, op.Tool, a.Index, a.Size)
		}
	}
	return head.ID, err
}

func uniqueID(p *project.Project, id string) error {
	if id == "" {
		return fmt.Errorf("对象 ID 不能为空")
	}
	for _, t := range p.Tables {
		if t.ID == id {
			return fmt.Errorf("对象 ID 已存在：%s", id)
		}
	}
	for _, t := range p.Images {
		if t.ID == id {
			return fmt.Errorf("对象 ID 已存在：%s", id)
		}
	}
	return nil
}
