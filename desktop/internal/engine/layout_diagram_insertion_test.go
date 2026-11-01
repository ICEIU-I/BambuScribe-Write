package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"reflect"
	"strings"
	"testing"
)

func TestDiagramInsertionReservesFrameAndPushesFollowingBody(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	text := "上方正文\n\n\n四、实验内容及步骤\n" + strings.Repeat("下方正文保持完整。\n", 30)
	plain, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, y := range []float64{15, 210} {
		im := ImageAsset{ID: "inserted", Data: wrappingImage(t).Data, X: 10, Y: y, Width: 100, Height: 90, Diagram: &diagram.Scene{
			Width: 100, Height: 90,
			Strokes: []diagram.Stroke{{ID: "line", Points: []diagram.Point{{X: 10, Y: 10}, {X: 90, Y: 10}}}},
		}}
		doc, err := f.LayoutWithImages(text, s, []ImageAsset{im})
		if err != nil {
			t.Fatal(err)
		}
		position := doc.ObjectLayout[0]
		if y == 15 && (position.Page != 0 || position.Y != y) {
			t.Fatal("following body displaced the insertion", position)
		}
		if y == 210 && (position.Page != 1 || position.Y != s.MarginTop) {
			t.Fatal("paper overflow did not carry the image forward", position)
		}
		var restored strings.Builder
		seen := map[int]bool{}
		for pi, page := range doc.Pages {
			restored.WriteString(page.Text)
			for _, cell := range page.Cells {
				if seen[cell.Start] {
					t.Fatal("duplicated body character", cell)
				}
				seen[cell.Start] = true
				if pi == position.Page && cell.Y < position.Y+im.Height+objectLayoutGap {
					if y == 15 && cell.Start < 4 {
						continue
					}
					t.Fatal("body entered the reserved image frame", cell)
				}
				if y == 210 && pi == 0 && cell.Y+cell.Height > y {
					t.Fatal("following body stayed ahead of the overflowing image", cell)
				}
			}
			if _, err := ExportPage(doc, pi, testWriting()); err != nil {
				t.Fatal(err)
			}
		}
		if restored.String() != text || !reflect.DeepEqual(doc.Pages[0].Cells[:4], plain.Pages[0].Cells[:4]) {
			t.Fatal("insertion changed preceding text or lost body content")
		}
		again, err := f.LayoutWithImages(text, s, []ImageAsset{im})
		if err != nil || !reflect.DeepEqual(doc, again) {
			t.Fatal("reloading changed insertion flow", err)
		}
	}
}
