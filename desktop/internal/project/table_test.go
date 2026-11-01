package project

import (
	"archive/zip"
	"bambuscribe/desktop/internal/engine"
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestProjectRetainsEditableTablesAndReadsVersionOne(t *testing.T) {
	p := testProject(t)
	p.Tables = []engine.TableAsset{{ID: "table", Page: 2, X: 10, Y: 20, ColumnWidths: []float64{30, 40}, RowHeights: []float64{10, 12}, Cells: []engine.TableCell{{Row: 0, Col: 0, RowSpan: 1, ColSpan: 2, Text: "合并标题"}, {Row: 1, Col: 0, RowSpan: 1, ColSpan: 1, Text: "实验数据"}, {Row: 1, Col: 1, RowSpan: 1, ColSpan: 1, Text: "10⁻¹⁹"}}, Style: "three-line", FontSize: 4, Padding: 1, Align: "center", VerticalAlign: "middle"}}
	raw, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(raw)
	if err != nil || !reflect.DeepEqual(p.Tables, restored.Tables) {
		t.Fatal("table topology changed across project round trip", err)
	}
	entries := unpack(t, raw)
	var manifest map[string]any
	if err = json.Unmarshal(entries[manifestName], &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["version"] = 1
	delete(manifest, "tables")
	entries[manifestName], _ = json.Marshal(manifest)
	var b bytes.Buffer
	out := zip.NewWriter(&b)
	for name, body := range entries {
		if err = writeEntry(out, name, body, zip.Deflate); err != nil {
			t.Fatal(err)
		}
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	legacy, err := Decode(b.Bytes())
	if err != nil || len(legacy.Tables) != 0 || legacy.Text != p.Text || !reflect.DeepEqual(legacy.Images, p.Images) {
		t.Fatal("legacy v1 project failed", err)
	}
	p.Tables[0].Cells[0].ColSpan = 3
	if _, err = Encode(p); err == nil {
		t.Fatal("invalid merge range was saved")
	}
}
