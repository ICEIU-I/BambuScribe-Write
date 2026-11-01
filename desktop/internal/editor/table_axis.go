package editor

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

func axisTable(t *engine.TableAsset, tool string, index int, size *float64) error {
	row, remove := strings.Contains(tool, ".row."), strings.HasSuffix(tool, ".delete")
	sizes, limit := &t.ColumnWidths, 20
	if row {
		sizes, limit = &t.RowHeights, 50
	}
	n := len(*sizes)
	if index < 0 || index > n || remove && index == n {
		return fmt.Errorf("行列索引无效，插入范围为 0–%d，删除范围为 0–%d", n, n-1)
	}
	if remove && n == 1 {
		return fmt.Errorf("至少保留一行一列，请用 object.delete 删除整个表格")
	}
	if !remove && n == limit {
		return fmt.Errorf("超过表格行列上限")
	}
	if remove {
		*sizes = append((*sizes)[:index], (*sizes)[index+1:]...)
	} else {
		v := (*sizes)[min(index, n-1)]
		if size != nil {
			v = *size
		}
		*sizes = append(*sizes, 0)
		copy((*sizes)[index+1:], (*sizes)[index:])
		(*sizes)[index] = v
	}
	var out []engine.TableCell
	for _, c := range t.Cells {
		at, span := &c.Col, &c.ColSpan
		if row {
			at, span = &c.Row, &c.RowSpan
		}
		if remove {
			if *at == index && *span == 1 {
				continue
			}
			if *at > index {
				*at--
			} else if *at+*span > index {
				*span--
			}
		} else {
			if *at >= index {
				*at++
			} else if *at+*span > index {
				*span++
			}
		}
		out = append(out, c)
	}
	t.Cells = out
	fillTable(t)
	return nil
}

func pasteTable(t *engine.TableAsset, raw json.RawMessage) error {
	var a struct {
		ID   string     `json:"id"`
		Row  int        `json:"row"`
		Col  int        `json:"col"`
		Data [][]string `json:"data"`
		TSV  *string    `json:"tsv"`
		File string     `json:"file"`
	}
	if err := Decode(raw, &a); err != nil {
		return err
	}
	sources := 0
	if a.Data != nil {
		sources++
	}
	if a.TSV != nil {
		sources++
	}
	if a.File != "" {
		sources++
	}
	if sources != 1 {
		return fmt.Errorf("粘贴须指定 data、tsv 或 file 其中一个")
	}
	if a.File != "" {
		b, err := ReadFile(a.File, 4<<20)
		if err != nil {
			return err
		}
		if len(b) > 4<<20 {
			return fmt.Errorf("TSV 文件超过 4 MB")
		}
		s := string(b)
		a.TSV = &s
	}
	if a.TSV != nil {
		r := csv.NewReader(strings.NewReader(*a.TSV))
		r.Comma = '\t'
		r.FieldsPerRecord = -1
		var err error
		a.Data, err = r.ReadAll()
		if err != nil {
			return fmt.Errorf("TSV 无法读取：%w", err)
		}
	}
	nr, nc := len(a.Data), 0
	for _, r := range a.Data {
		nc = max(nc, len(r))
	}
	if a.Row < 0 || a.Col < 0 || nr == 0 || nc == 0 || a.Row+nr > 50 || a.Col+nc > 20 {
		return fmt.Errorf("粘贴后的表格须在 50 行、20 列内")
	}
	for len(t.RowHeights) < a.Row+nr {
		if err := axisTable(t, "table.row.insert", len(t.RowHeights), nil); err != nil {
			return err
		}
	}
	for len(t.ColumnWidths) < a.Col+nc {
		if err := axisTable(t, "table.column.insert", len(t.ColumnWidths), nil); err != nil {
			return err
		}
	}
	_, indices, err := selected(t, Selection{a.Row, a.Col, a.Row + nr - 1, a.Col + nc - 1})
	if err != nil {
		return err
	}
	splitIndices(t, indices)
	for r := 0; r < nr; r++ {
		for c := 0; c < nc; c++ {
			text := ""
			if c < len(a.Data[r]) {
				text = a.Data[r][c]
			}
			t.Cells[cellIndex(t, a.Row+r, a.Col+c)].Text = text
		}
	}
	return nil
}
