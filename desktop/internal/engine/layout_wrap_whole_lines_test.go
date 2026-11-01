package engine

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestBodySkipsObjectSlotsAtBothEdgesAndBetweenObjects(t *testing.T) {
	f := testFont(t)
	for _, sizing := range [][3]float64{{6, 100, 0}, {8.2, 76.5, 0}, {4, 200, 0.5}} {
		for _, cells := range []int{1, 2, 3, 5, 10} {
			for _, side := range []string{"left", "middle", "right"} {
				t.Run(fmt.Sprintf("%v/%d/%s", sizing, cells, side), func(t *testing.T) {
					s := DefaultLayout()
					s.FontSize, s.LetterSpacing, s.LetterGap = sizing[0], sizing[1], sizing[2]
					s.MarginLeft, s.MarginRight = 10, 10
					left, right := s.MarginLeft, s.PaperWidth-s.MarginRight
					width := float64(cells)*(s.FontSize*s.LetterSpacing/100+s.LetterGap) - s.LetterGap
					x := left
					if side == "middle" {
						x = (left + right - width) / 2
					} else if side == "right" {
						x = right - width
					}
					var objects []RegionBounds
					if side != "left" {
						objects = append(objects, RegionBounds{0, 0, x - objectLayoutGap, 40})
					}
					if side != "right" {
						objects = append(objects, RegionBounds{x + width + objectLayoutGap, 0, s.PaperWidth - x - width - objectLayoutGap, 40})
					}
					for _, text := range []string{"速率为V，载流子受到磁场作用。", "v=9.8m/s", "你"} {
						doc, err := f.layoutText(text, s, map[int][]RegionBounds{0: objects})
						if err != nil {
							t.Fatal(err)
						}
						if doc.Pages[0].Text != text || len(doc.Pages[0].Cells) != len([]rune(text)) {
							t.Fatal("skipping a narrow slot lost text", text)
						}
						for _, c := range doc.Pages[0].Cells {
							if c.Y < 41 {
								t.Fatal("body text squeezed into a narrow object slot", c)
							}
						}
					}
				})
			}
		}
	}
}

func TestNarrowImageGapPreservesTextPagesObjectsAndExport(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	left, right := wrappingImage(t), wrappingImage(t)
	left.ID, left.Page, left.Width, left.Height = "left", 1, 100, 40
	right.ID, right.Page, right.X, right.Width, right.Height = "right", 1, 110, 53, 40
	images := []ImageAsset{left, right}
	for _, height := range []float64{40, s.PaperHeight} {
		images[0].Height, images[1].Height = height, height
		text := "标题\f速率为V，载流子受到磁场作用。\n正文继续。"
		doc, err := f.LayoutWithImages(text, s, images)
		if err != nil {
			t.Fatal(err)
		}
		assertBodyExcludesObjects(t, doc)
		var restored strings.Builder
		for i, page := range doc.Pages {
			restored.WriteString(page.Text)
			if len(page.Strokes) > 0 {
				if _, err := ExportPage(doc, i, testWriting()); err != nil {
					t.Fatal(err)
				}
			}
		}
		if restored.String() != text {
			t.Fatal("wrapping changed text or explicit page boundaries")
		}
		for _, c := range doc.Pages[1].Cells {
			if c.Y < height+objectLayoutGap {
				t.Fatal("text still occupies the gap between images", c)
			}
		}
		if height == s.PaperHeight && (len(doc.Pages) != 3 || doc.Pages[2].Cells[0].Y != 0) {
			t.Fatal("blocked page did not resume horizontal text on the next sheet")
		}
		for i, p := range doc.ObjectLayout {
			if p.Page != images[i].Page || p.X != images[i].X || p.Y != images[i].Y {
				t.Fatal("text wrapping moved an image", p)
			}
		}
	}
}

func TestWholeLineAvoidanceKeepsNarrowPaperAndObjectsOutsideBody(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize, s.MarginRight = 6, 40
	plain, err := f.Layout("你好", s)
	if err != nil {
		t.Fatal(err)
	}
	objects := map[int][]RegionBounds{0: {{130, 0, 20, 40}}}
	doc, err := f.layoutText("你好", s, objects)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc.Pages[0].Cells, plain.Pages[0].Cells) {
		t.Fatal("an object outside the body displaced text")
	}
	s.PaperWidth, s.MarginRight = 20, 0
	doc, err = f.Layout("你好你好你好你好", s)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Cells[0].Y != 0 || len(doc.Pages[0].Cells) != 8 || len(doc.Pages) != 1 {
		t.Fatal("unobstructed narrow paper was incorrectly excluded")
	}
}
