package engine

import (
	"archive/zip"
	"bytes"
	"fmt"
)

func ExportPage(d *Document, index int, w WritingSettings) ([]byte, error) {
	if index < 0 || index >= len(d.Pages) {
		return nil, fmt.Errorf("页码不存在")
	}
	gcode, err := GenerateGCode(d.Pages[index], d.Settings, w)
	if err != nil {
		return nil, err
	}
	// Like Chixie's A1 uploader, send UTF-8 G-code directly. A sliced 3MF
	// describes an extrusion job and makes the local launcher request filament.
	return []byte(gcode), nil
}

func ExportAll(d *Document, w WritingSettings) ([]byte, error) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	count := 0
	for i := range d.Pages {
		// Preserve page numbering but don't produce executable empty-page jobs.
		if len(d.Pages[i].Strokes) == 0 && len(d.Pages[i].Missing) == 0 && len(d.Pages[i].Issues) == 0 {
			continue
		}
		data, err := ExportPage(d, i, w)
		if err != nil {
			return nil, err
		}
		entry, err := z.Create(fmt.Sprintf("page-%03d.gcode", i+1))
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write(data); err != nil {
			return nil, err
		}
		count++
	}
	if count == 0 {
		return nil, fmt.Errorf("文档没有可书写笔画")
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
