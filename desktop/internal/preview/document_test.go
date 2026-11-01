package preview

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/json"
	"reflect"
	"testing"
)

func TestPackedPreviewRestoresEveryVectorWithoutChangingTheDocument(t *testing.T) {
	a := []engine.Stroke{{{X: 1.2345678901234567, Y: 2}, {X: 3, Y: 4.987654321098765}}}
	b := []engine.Stroke{{{X: 5, Y: 6}, {X: 7, Y: 8}}}
	doc := &engine.Document{Pages: []engine.Page{{
		Strokes: append(append([]engine.Stroke{}, a...), b...), TextStrokes: a,
		Issues: []string{"行距过小"}, Warnings: []string{"识别提示"},
		Diagnostics: []engine.LayoutDiagnostic{{Code: "line-overlap", Message: "行距过小", RegionBounds: engine.RegionBounds{X: 1, Y: 2, Width: 3, Height: 4}}},
		Images: []engine.ImagePlacement{{
			Strokes: a, GraphicStrokes: a, SourceStrokes: b,
			TextRegions: []engine.ImageTextRegion{{Strokes: a, SourceStrokes: b}},
		}},
		Tables: []engine.TablePlacement{{Strokes: b, Cells: []engine.TableCellPlacement{{Strokes: b}}}},
		Ink:    []engine.InkLayer{{ObjectID: "image", Part: "graphic", Strokes: b}, {Part: "0", Strokes: a}},
	}}}
	before, _ := json.Marshal(doc)
	packed := Pack(doc)
	if len(packed.Strokes) != 2 {
		t.Fatal("repeated vectors were transferred again", len(packed.Strokes))
	}
	raw, err := json.Marshal(packed)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Document any
		Strokes  [][]float64
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	var restore func(any) any
	restore = func(value any) any {
		switch v := value.(type) {
		case []any:
			for i := range v {
				v[i] = restore(v[i])
			}
		case map[string]any:
			for key, field := range v {
				if key == "strokes" || key == "textStrokes" || key == "sourceStrokes" || key == "graphicStrokes" {
					if refs, ok := field.([]any); ok {
						strokes := []any{}
						for _, ref := range refs {
							coordinates := wire.Strokes[int(ref.(float64))]
							points := []any{}
							for i := 0; i < len(coordinates); i += 2 {
								points = append(points, map[string]any{"x": coordinates[i], "y": coordinates[i+1]})
							}
							strokes = append(strokes, points)
						}
						v[key] = strokes
					}
				} else {
					v[key] = restore(field)
				}
			}
		}
		return value
	}
	var expected any
	json.Unmarshal(before, &expected)
	if got := restore(wire.Document); !reflect.DeepEqual(got, expected) {
		t.Fatalf("packed preview changed geometry or metadata:\n%#v\n%#v", got, expected)
	}
	after, _ := json.Marshal(doc)
	if string(before) != string(after) {
		t.Fatal("packing changed the cached export document")
	}
	if len(raw) >= len(before) {
		t.Fatal("packing did not reduce repeated geometry")
	}
}

func TestPackedPreviewPreservesEmptyPagesAndStrokeLists(t *testing.T) {
	doc := &engine.Document{Pages: []engine.Page{{Strokes: []engine.Stroke{}, Images: []engine.ImagePlacement{}}}}
	raw, err := json.Marshal(Pack(doc))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Document struct {
			Pages []struct {
				Strokes     []int
				TextStrokes []int
				Images      []any
			}
		}
	}
	json.Unmarshal(raw, &decoded)
	p := decoded.Document.Pages[0]
	if p.Strokes == nil || p.TextStrokes != nil || p.Images == nil {
		t.Fatal("packing changed nil versus empty lists")
	}
}
