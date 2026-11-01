package engine

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

const maxTables = 50

func (t TableAsset) Validate() error {
	if t.Kind != "" && !t.isTextBox() {
		return fmt.Errorf("未知对象类型，请重新插入表格或文本框")
	}
	if t.isTextBox() && (len(t.RowHeights) != 1 || len(t.ColumnWidths) != 1 || len(t.Cells) != 1 || t.Style != "none" || t.Cells[0].Borders != nil) {
		return fmt.Errorf("文本框结构无效，请重新插入文本框")
	}
	if len(t.GroupID) > 128 || t.ID == "" || t.Page < 0 || t.Page > maxDocumentCharacters {
		return fmt.Errorf("表格标识或页码无效，请重新插入表格")
	}
	if len(t.RowHeights) < 1 || len(t.RowHeights) > 50 || len(t.ColumnWidths) < 1 || len(t.ColumnWidths) > 20 {
		return fmt.Errorf("表格须为 1–50 行、1–20 列")
	}
	if !finite(t.X) || !finite(t.Y) || t.X < 0 || t.Y < 0 || !finite(t.FontSize) || t.FontSize < 1 || t.FontSize > 40 || !finite(t.Padding) || t.Padding < 0 || t.Padding > 10 {
		return fmt.Errorf("表格位置、字号或内边距无效，请检查表格设置")
	}
	for _, sizes := range [][]float64{t.RowHeights, t.ColumnWidths} {
		for _, v := range sizes {
			if !finite(v) || v < 3 || v > 1000 {
				return fmt.Errorf("表格行高和列宽须为 3–1000 mm")
			}
		}
	}
	if t.Style != "grid" && t.Style != "three-line" && t.Style != "horizontal" && t.Style != "none" {
		return fmt.Errorf("未知的表格样式，请重新选择样式")
	}
	if !tableAlign(t.Align, false) || !tableAlign(t.VerticalAlign, true) {
		return fmt.Errorf("未知的表格对齐方式")
	}
	if len(t.Cells) > len(t.RowHeights)*len(t.ColumnWidths) {
		return fmt.Errorf("表格单元格重复，请重新插入表格")
	}
	owners := make([]bool, len(t.RowHeights)*len(t.ColumnWidths))
	characters := 0
	for _, c := range t.Cells {
		if c.Row < 0 || c.Col < 0 || c.RowSpan < 1 || c.ColSpan < 1 || c.Row+c.RowSpan > len(t.RowHeights) || c.Col+c.ColSpan > len(t.ColumnWidths) || !tableAlign(c.Align, false) || !tableAlign(c.VerticalAlign, true) {
			return fmt.Errorf("表格合并范围或对齐方式无效")
		}
		if !utf8.ValidString(c.Text) {
			return fmt.Errorf("表格文字编码无效")
		}
		characters += utf8.RuneCountInString(c.Text)
		if characters > maxDocumentCharacters {
			return fmt.Errorf("表格文字最多 20,000 字，请拆分表格")
		}
		for _, r := range c.Text {
			if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
				return fmt.Errorf("表格文字包含不可见控制字符")
			}
		}
		for row := c.Row; row < c.Row+c.RowSpan; row++ {
			for col := c.Col; col < c.Col+c.ColSpan; col++ {
				i := row*len(t.ColumnWidths) + col
				if owners[i] {
					return fmt.Errorf("表格合并区域重叠")
				}
				owners[i] = true
			}
		}
	}
	for _, owned := range owners {
		if !owned {
			return fmt.Errorf("表格缺少单元格，请重新插入表格")
		}
	}
	return nil
}

func tableAlign(a string, vertical bool) bool {
	if vertical {
		return a == "" || a == "top" || a == "middle" || a == "bottom"
	}
	return a == "" || a == "left" || a == "center" || a == "right"
}

func ValidateTables(tables []TableAsset) error {
	if len(tables) > maxTables {
		return fmt.Errorf("表格和文本框共最多 50 个，请拆分文档")
	}
	seen := map[string]bool{}
	total := 0
	for _, t := range tables {
		if seen[t.ID] {
			return fmt.Errorf("表格标识重复")
		}
		seen[t.ID] = true
		if err := t.Validate(); err != nil {
			return err
		}
		for _, c := range t.Cells {
			total += utf8.RuneCountInString(c.Text)
		}
	}
	if total > maxDocumentCharacters {
		return fmt.Errorf("表格文字共超过 20,000 字，请拆分文档")
	}
	return nil
}
