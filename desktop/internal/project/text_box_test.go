package project

import (
	"bambuscribe/desktop/internal/engine"
	"reflect"
	"testing"
)

func TestProjectRetainsIndependentTextBox(t *testing.T) {
	p := testProject(t)
	p.Tables = []engine.TableAsset{{ID: "caption", Kind: "textbox", Page: 3, X: 15, Y: 10, ColumnWidths: []float64{70}, RowHeights: []float64{9}, Cells: []engine.TableCell{{Row: 0, Col: 0, RowSpan: 1, ColSpan: 1, Text: "表1 I-U关系\n10⁻¹¹ A"}}, Style: "none", FontSize: 5, Padding: .5, Align: "center", VerticalAlign: "middle"}}
	data, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(data)
	if err != nil || !reflect.DeepEqual(p.Tables, restored.Tables) || !reflect.DeepEqual(p.Images, restored.Images) {
		t.Fatal("text box was lost or damaged image objects in the project", err)
	}
}
