package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"github.com/google/uuid"
)

func textBox(f *engine.Font, text string, settings engine.LayoutSettings, width float64, align string) (engine.TableAsset, error) {
	text = cellText(text)
	size, minimum := settings.FontSize, importMinimumSize(settings.FontSize)
	measure := func(text string, size float64) float64 {
		w, _ := f.MeasureTextBoxText(text, size, settings)
		return w
	}
	if measure(text, size) > width-1 {
		if measure(text, minimum) > width-1 {
			wrapped, err := wrapText(text, width-1, func(text string) float64 { return measure(text, minimum) })
			if err != nil {
				return engine.TableAsset{}, err
			}
			text, size = wrapped, minimum
		} else {
			low, high := minimum, size
			for range 40 {
				mid := (low + high) / 2
				if measure(text, mid) <= width-1 {
					low = mid
				} else {
					high = mid
				}
			}
			size = low
		}
	}
	_, h := f.MeasureTextBoxText(text, size, settings)
	if text != "" {
		h = max(h, size)
	}
	return engine.TableAsset{ID: "import-title-" + uuid.NewString(), Kind: "textbox", ColumnWidths: []float64{width}, RowHeights: []float64{max(3, h+1)}, Cells: []engine.TableCell{{RowSpan: 1, ColSpan: 1, Text: text}}, Style: "none", FontSize: size, Padding: .5, Align: align, VerticalAlign: "middle"}, nil
}

func readableImportWidth(f *engine.Font, t Table, settings engine.LayoutSettings) float64 {
	minimum := importMinimumSize(settings.FontSize)
	widths := make([]float64, len(t.Cells))
	for i, c := range t.Cells {
		widths[i], _ = f.MeasureTableText(cellText(c.Text), 1)
	}
	width := sum(minimumColumns(t, widths, minimum, 1))
	for _, text := range append([]string{t.Title}, t.Notes...) {
		w, _ := f.MeasureTextBoxText(cellText(text), minimum, settings)
		width = max(width, w+1)
	}
	return width
}
