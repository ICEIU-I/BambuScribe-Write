package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"fmt"
	"github.com/google/uuid"
)

func sizeTable(f *engine.Font, t Table, size, width float64) ([]engine.TableAsset, []string, error) {
	pad := 1.0
	requestedSize := size
	size, columns, err := fitTableColumns(f, t, size, width, pad)
	if err != nil {
		return nil, nil, err
	}
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
	minimum := sum(a.ColumnWidths)
	ratio := sum(t.Columns)
	for j, v := range t.Columns {
		a.ColumnWidths[j] += (width - minimum) * v / ratio
	}
	for _, spanning := range []bool{false, true} {
		for _, c := range t.Cells {
			if (c.RowSpan > 1) != spanning {
				continue
			}
			_, h := f.MeasureTableText(cellText(c.Text), size)
			if cellText(c.Text) != "" {
				// Preserve the intended physical row height now that measurement
				// excludes unused font margins. Short digits must grow to fit the
				// row instead of making newly imported rows tiny again.
				h = max(h, size)
			}
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
	out := []engine.TableAsset{}
	if t.Title != "" {
		title, err := textBox(f, t.Title, requestedSize, width, "center")
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
			b, err := textBox(f, note, requestedSize, width, "left")
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
func textBox(f *engine.Font, text string, size, width float64, align string) (engine.TableAsset, error) {
	text = cellText(text)
	size, err := fitTextBoxFont(f, text, size, width)
	if err != nil {
		return engine.TableAsset{}, err
	}
	_, h := f.MeasureTableText(text, size)
	if text != "" {
		h = max(h, size)
	}
	return engine.TableAsset{ID: "import-title-" + uuid.NewString(), Kind: "textbox", ColumnWidths: []float64{width}, RowHeights: []float64{max(3, h+1)}, Cells: []engine.TableCell{{RowSpan: 1, ColSpan: 1, Text: text}}, Style: "none", FontSize: size, Padding: .5, Align: align, VerticalAlign: "middle"}, nil
}
func sum(a []float64) (v float64) {
	for _, n := range a {
		v += n
	}
	return
}
