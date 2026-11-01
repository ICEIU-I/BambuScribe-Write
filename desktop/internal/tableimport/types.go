// Package tableimport turns recognized table topology into native editable
// objects. Recognition never controls document mutations or paper coordinates.
package tableimport

import (
	"bambuscribe/desktop/internal/editor"
	"bambuscribe/desktop/internal/engine"
	"fmt"
	"math"
	"strings"
)

type Cell struct {
	Row       int                  `json:"row"`
	Col       int                  `json:"col"`
	RowSpan   int                  `json:"rowSpan"`
	ColSpan   int                  `json:"colSpan"`
	Text      string               `json:"text"`
	Uncertain bool                 `json:"uncertain"`
	Borders   *engine.TableBorders `json:"borders"`
}
type Table struct {
	Title   string    `json:"title"`
	Notes   []string  `json:"notes"`
	Rows    int       `json:"rows"`
	Columns []float64 `json:"columns"`
	Style   string    `json:"style"`
	Cells   []Cell    `json:"cells"`
}
type Extraction struct {
	Tables []Table `json:"tables"`
	Review *Review `json:"review,omitempty"`
}

// Review is accepted only for compatibility with previously saved CLI extractions.
// New recognition results omit it.
type Review struct {
	Reviewed bool     `json:"reviewed"`
	Changes  []string `json:"changes"`
	Issues   []string `json:"issues"`
}
type Placement struct {
	Page      int        `json:"page"` // 1-based, as in the CLI
	Y         float64    `json:"y"`
	AfterID   string     `json:"afterId,omitempty"`
	AtEnd     bool       `json:"atEnd,omitempty"`
	Positions []Position `json:"positions,omitempty"`
}

// Positions are in paper millimeters and follow the extraction's table order.
// Each table moves together with its title and notes during import planning.
type Position struct {
	Page  int     `json:"page"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Width float64 `json:"width"`
}
type Group struct {
	Position
	Index     int      `json:"index"`
	Title     string   `json:"title"`
	Height    float64  `json:"height"`
	ObjectIDs []string `json:"objectIds"`
}
type Plan struct {
	Objects  []engine.TableAsset     `json:"objects"`
	Warnings []string                `json:"warnings"`
	Pages    []int                   `json:"pages"`
	Groups   []Group                 `json:"groups"`
	Preview  []engine.TablePlacement `json:"preview"`
}

func Decode(raw []byte) (Extraction, error) {
	var e Extraction
	if err := editor.Decode(raw, &e); err != nil {
		return e, err
	}
	e.normalizeText()
	return e, e.Validate()
}
func (e Extraction) Validate() error {
	if len(e.Tables) < 1 || len(e.Tables) > 10 {
		return fmt.Errorf("图片中须包含 1–10 个表格，请裁剪后重试")
	}
	chars := 0
	for i, t := range e.Tables {
		if t.Rows < 1 || t.Rows > 50 || len(t.Columns) < 1 || len(t.Columns) > 20 || len(t.Notes) > 10 {
			return fmt.Errorf("表格 %d 结构过大，请裁剪后重试", i+1)
		}
		for _, v := range t.Columns {
			if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 10000 {
				return fmt.Errorf("列宽比例无效")
			}
		}
		a := engine.TableAsset{ID: "validation", Style: t.Style, FontSize: 5, Align: "center", VerticalAlign: "middle", Padding: 1}
		for range t.Rows {
			a.RowHeights = append(a.RowHeights, 10)
		}
		for range t.Columns {
			a.ColumnWidths = append(a.ColumnWidths, 10)
		}
		for _, c := range t.Cells {
			a.Cells = append(a.Cells, engine.TableCell{Row: c.Row, Col: c.Col, RowSpan: c.RowSpan, ColSpan: c.ColSpan, Text: c.Text, Borders: c.Borders})
			chars += len([]rune(c.Text))
		}
		if err := a.Validate(); err != nil {
			return fmt.Errorf("表格 %d：%w", i+1, err)
		}
		chars += len([]rune(t.Title))
		for _, n := range t.Notes {
			chars += len([]rune(n))
		}
	}
	if chars > 20000 {
		return fmt.Errorf("导入内容超过 20,000 字，请拆分图片")
	}
	return nil
}
func cellText(s string) string {
	s = normalizeText(s)
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = strings.NewReplacer("–", "-", "‑", "-").Replace(s)
	if strings.Trim(strings.TrimSpace(s), ".…⋯ ") == "" {
		return ""
	}
	return s
}
