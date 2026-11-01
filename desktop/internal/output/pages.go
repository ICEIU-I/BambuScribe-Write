// Package output performs the shared preflight before desktop or CLI uploads.
package output

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/printer"
	"fmt"
)

// PreparePages validates the entire batch before any connection or transfer.
// Indices are zero-based, sorted and unique; nil selects every nonempty page.
func PreparePages(d *engine.Document, w engine.WritingSettings, indices ...int) ([]printer.PageUpload, error) {
	if len(indices) == 0 {
		for i := range d.Pages {
			indices = append(indices, i)
		}
	}
	pages := make([]printer.PageUpload, 0, len(d.Pages))
	previous := -1
	for _, i := range indices {
		if i <= previous || i < 0 || i >= len(d.Pages) {
			return nil, fmt.Errorf("页码无效，请重新选择上传页面")
		}
		previous = i
		p := d.Pages[i]
		if len(p.Strokes) == 0 && len(p.Missing) == 0 && len(p.Issues) == 0 {
			continue
		}
		data, err := engine.ExportPage(d, i, w)
		if err != nil {
			return nil, fmt.Errorf("第 %d 页无法上传：%w", i+1, err)
		}
		pages = append(pages, printer.PageUpload{Page: i + 1, Data: data})
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("文档没有可书写笔画")
	}
	return pages, nil
}
