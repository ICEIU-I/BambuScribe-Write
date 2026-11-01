package engine

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestOneShotCutsFollowTextBoxVectorsAndIgnoreNewText(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	box := sampleTextBox()
	box.Page = 0
	box.X = 20
	box.Y = 40
	for _, free := range []bool{false, true} {
		er := PageErasure{ID: "cut", X: box.X, Y: box.Y, Width: 38, Height: 20}
		if free {
			er.Path = []Point{{box.X + 35, box.Y}, {box.X + 35, box.Y + 20}}
			er.Radius = 3
		}
		first, err := f.LayoutWithErasures("", s, nil, []TableAsset{box}, []PageErasure{er})
		if err != nil {
			t.Fatal(err)
		}
		if len(*first.Erasures[0].Cuts) == 0 {
			t.Fatal("fixture not erased")
		}
		moved := box
		moved.X += 35
		moved.Y += 20
		moved.Page = 1
		got, err := f.LayoutWithErasures("", s, nil, []TableAsset{moved}, first.Erasures)
		if err != nil {
			t.Fatal(err)
		}
		want := []Stroke{}
		for _, st := range first.Pages[0].Strokes {
			n := Stroke{}
			for _, p := range st {
				n = append(n, Point{p.X + 35, p.Y + 20})
			}
			want = append(want, n)
		}
		closeInk(t, got.Pages[1].Strokes, want)
		moved.ColumnWidths = []float64{box.ColumnWidths[0] * 1.5}
		moved.RowHeights = []float64{box.RowHeights[0] * 1.5}
		moved.Padding *= 1.5
		resized, err := f.LayoutWithErasures("", s, nil, []TableAsset{moved}, first.Erasures)
		if err != nil {
			t.Fatal(err)
		}
		scaled := []Stroke{}
		for _, st := range first.Pages[0].Strokes {
			n := Stroke{}
			for _, p := range st {
				n = append(n, Point{moved.X + (p.X-box.X)*1.5, moved.Y + (p.Y-box.Y)*1.5})
			}
			scaled = append(scaled, n)
		}
		closeInk(t, resized.Pages[1].Strokes, scaled)
		moved.Cells = append([]TableCell(nil), box.Cells...)
		moved.Cells[0].Text = "新的完整图注"
		changed, err := f.LayoutWithErasures("", s, nil, []TableAsset{moved}, first.Erasures)
		if err != nil {
			t.Fatal(err)
		}
		plain, _ := f.LayoutWithObjects("", s, nil, []TableAsset{moved})
		if !reflect.DeepEqual(changed.Pages[1].Strokes, plain.Pages[1].Strokes) {
			t.Fatal("new text inherited erased vectors")
		}
		raw, _ := json.Marshal(first.Erasures)
		var saved []PageErasure
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		reopened, _ := f.LayoutWithErasures("", s, nil, []TableAsset{box}, saved)
		if !reflect.DeepEqual(first.Pages[0].Strokes, reopened.Pages[0].Strokes) {
			t.Fatal("save/reload changed erased ink")
		}
	}
}
func closeInk(t *testing.T, a, b []Stroke) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("stroke count %d != %d", len(a), len(b))
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			t.Fatal("vertex count changed")
		}
		for j := range a[i] {
			if math.Hypot(a[i][j].X-b[i][j].X, a[i][j].Y-b[i][j].Y) > 1e-7 {
				t.Fatal("cut moved on source", a[i][j], b[i][j])
			}
		}
	}
}
func TestOneShotImageCutsRotateScaleAndMoveWithImage(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	im := ImageAsset{ID: "image", Data: testImageData(t), X: 20, Y: 30, Width: 40, Height: 30}
	first, err := f.LayoutWithErasures("", s, []ImageAsset{im}, nil, []PageErasure{{ID: "cut", X: 20, Y: 30, Width: 25, Height: 20}})
	if err != nil {
		t.Fatal(err)
	}
	if len(*first.Erasures[0].Cuts) == 0 {
		t.Fatal("empty fixture")
	}
	old := im
	im.X = 60
	im.Y = 80
	im.Width *= 1.5
	im.Height *= 1.5
	im.Rotation = 30
	got, err := f.LayoutWithErasures("", s, []ImageAsset{im}, nil, first.Erasures)
	if err != nil {
		t.Fatal(err)
	}
	want := []Stroke{}
	for _, st := range first.Pages[0].Strokes {
		n := Stroke{}
		for _, p := range st {
			n = append(n, imagePoint(im, im.X+(p.X-old.X)*1.5, im.Y+(p.Y-old.Y)*1.5))
		}
		want = append(want, n)
	}
	closeInk(t, got.Pages[0].Strokes, want)
}
func TestOneShotBodyCutsFollowReflowAndEmptyCutDoesNotErase(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	first, err := f.LayoutWithErasures("正文测试", s, nil, nil, []PageErasure{{ID: "cut", X: 0, Y: 0, Width: 6, Height: 6}})
	if err != nil {
		t.Fatal(err)
	}
	s.MarginTop = 30
	s.FontSize = 12
	moved, err := f.LayoutWithErasures("正文测试", s, nil, nil, first.Erasures)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Pages[0].Strokes) != len(first.Pages[0].Strokes) {
		t.Fatal("body erasure left source after reflow")
	}
	empty := []InkCut{}
	doc, _ := f.LayoutWithErasures("正文测试", s, nil, nil, []PageErasure{{ID: "empty", Width: 163, Height: 240, Cuts: &empty}})
	plain, _ := f.LayoutWithObjects("正文测试", s, nil, nil)
	if !reflect.DeepEqual(doc.Pages[0].Strokes, plain.Pages[0].Strokes) {
		t.Fatal("empty materialized gesture became permanent mask")
	}
}
