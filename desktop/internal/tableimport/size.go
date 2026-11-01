package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"fmt"
	"github.com/google/uuid"
)

func sizeTable(f *engine.Font, t Table, settings engine.LayoutSettings, width float64) ([]engine.TableAsset, []string, error) {
	pad := 1.0
	requestedSize := settings.FontSize
	t, size, columns, err := fitTableColumns(f, t, requestedSize, width, pad)
	if err != nil {
		return nil, nil, err
	}
	// Do not stretch a short table across the page: that used to enlarge
	// short digits far beyond the body size just to fill their cells.
	compactWidth := sum(columns)
	for _, caption := range append([]string{t.Title}, t.Notes...) {
		w, _ := f.MeasureTextBoxText(cellText(caption), requestedSize, settings)
		compactWidth = max(compactWidth, w+1)
	}
	width = min(width, compactWidth)
	a := engine.TableAsset{ID: "import-" + uuid.NewString(), Style: t.Style, FontSize: size, Padding: pad, Align: "center", VerticalAlign: "middle"}
	a.ColumnWidths = columns
	for range t.Rows {
		// Start at the editor minimum; measured text below supplies only the
		// height each row actually needs, including explicit lines and spans.
		a.RowHeights = append(a.RowHeights, 3)
	}
	warnings := []string{}
	if size < requestedSize {
		warnings = append(warnings, fmt.Sprintf("表格字号已自动缩小至 %.2f mm", size))
	}
	distributeColumns(a.ColumnWidths, t.Columns, width)
	for _, spanning := range []bool{false, true} {
		for _, c := range t.Cells {
			if (c.RowSpan > 1) != spanning {
				continue
			}
			_, h := f.MeasureTableText(cellText(c.Text), size)
			used := sum(a.RowHeights[c.Row : c.Row+c.RowSpan])
			if h+2*pad > used {
				a.RowHeights[c.Row+c.RowSpan-1] += h + 2*pad - used
			}
		}
	}
	for _, c := range t.Cells {
		a.Cells = append(a.Cells, engine.TableCell{Row: c.Row, Col: c.Col, RowSpan: c.RowSpan, ColSpan: c.ColSpan, Text: cellText(c.Text), Borders: c.Borders})
		if c.Uncertain {
			warnings = append(warnings, fmt.Sprintf("%s第 %d 行第 %d 列识别不确定，请对照图片修改", t.Title, c.Row+1, c.Col+1))
		}
	}
	a = f.CompactTable(a, settings)
	width = sum(a.ColumnWidths)
	out := []engine.TableAsset{}
	if t.Title != "" {
		title, err := textBox(f, t.Title, settings, width, "center")
		if err != nil {
			return nil, nil, err
		}
		out = append(out, title)
		if title.FontSize < requestedSize {
			warnings = append(warnings, fmt.Sprintf("表名字号已自动缩小至 %.2f mm", title.FontSize))
		}
	}
	for _, note := range t.Notes {
		if note != "" {
			b, err := textBox(f, note, settings, width, "left")
			if err != nil {
				return nil, nil, err
			}
			out = append(out, b)
			if b.FontSize < requestedSize {
				warnings = append(warnings, fmt.Sprintf("说明字号已自动缩小至 %.2f mm", b.FontSize))
			}
		}
	}
	out = append(out, a)
	return out, warnings, nil
}
func sum(a []float64) (v float64) {
	for _, n := range a {
		v += n
	}
	return
}
