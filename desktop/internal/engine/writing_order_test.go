package engine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
)

// Compare every output stroke with the original preview layers, including its
// point order. Merely checking the first travel move could miss dropped ink or
// a sorter that interleaves strokes from different characters/objects.
func assertWritingBlocks(t *testing.T, page Page, want []string) {
	t.Helper()
	sources := map[string][]Stroke{"body": page.TextStrokes}
	for _, im := range page.Images {
		sources[im.ID] = im.Strokes
	}
	for _, table := range page.Tables {
		for _, stroke := range table.Strokes {
			placed := make(Stroke, len(stroke))
			for i, p := range stroke {
				placed[i] = Point{p.X + table.X, p.Y + table.Y}
			}
			sources[table.ID] = append(sources[table.ID], placed)
		}
	}
	owners := map[string]string{}
	for name, strokes := range sources {
		for _, stroke := range strokes {
			data, _ := json.Marshal(stroke)
			owners[string(data)] = name
		}
	}
	next := map[string]int{}
	var got []string
	for _, stroke := range page.Strokes {
		data, _ := json.Marshal(stroke)
		name, ok := owners[string(data)]
		if !ok || next[name] >= len(sources[name]) || !reflect.DeepEqual(stroke, sources[name][next[name]]) {
			t.Fatal("writing changed, duplicated or reordered a source stroke", name, next[name])
		}
		next[name]++
		if len(got) == 0 || got[len(got)-1] != name {
			got = append(got, name)
		}
	}
	for name, strokes := range sources {
		if next[name] != len(strokes) {
			t.Fatal("writing lost strokes", name)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("writing blocks = %v, want %v", got, want)
	}
}

func assertWritingReplay(t *testing.T, doc *Document) {
	t.Helper()
	for _, mode := range []string{"manual", "auto"} {
		w := testWriting()
		w.OriginMode = mode
		data, err := ExportAll(doc, w)
		if err != nil {
			t.Fatal(err)
		}
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil || len(archive.File) != len(doc.Pages) {
			t.Fatal("missing exported pages", err)
		}
		for pi, page := range doc.Pages {
			single, err := ExportPage(doc, pi, w)
			if err != nil {
				t.Fatal(err)
			}
			r, err := archive.File[pi].Open()
			if err != nil {
				t.Fatal(err)
			}
			zipped, err := io.ReadAll(r)
			r.Close()
			if err != nil || !bytes.Equal(single, zipped) {
				t.Fatal("single-page and batch exports disagree", err)
			}
			got := replayWrittenStrokes(t, string(single), Point{Y: doc.Settings.PaperHeight}, *w.PenDown)
			if len(got) != len(page.Strokes) {
				t.Fatal("G-code lost strokes")
			}
			for i, stroke := range page.Strokes {
				if len(got[i]) != len(stroke) {
					t.Fatal("G-code lost points")
				}
				for j, p := range stroke {
					if math.Abs(got[i][j].X-p.X) > .00051 || math.Abs(got[i][j].Y-(doc.Settings.PaperHeight-p.Y)) > .00051 {
						t.Fatalf("%s page %d stroke %d: G-code changed writing order or coordinates", mode, pi+1, i)
					}
				}
			}
		}
	}
}

func TestWritingOrderInterleavesImagesWithBody(t *testing.T) {
	f := testFont(t)
	for _, font := range []string{FontHandwriting, FontDC} {
		for _, test := range []struct {
			name string
			y    float64
			want []string
		}{
			{"top", 0, []string{"image", "body"}},
			{"middle", 60, []string{"body", "image", "body"}},
			{"bottom", 210, []string{"body", "image"}},
		} {
			t.Run(font+"/"+test.name, func(t *testing.T) {
				s := DefaultLayout()
				s.FontID, s.FontSize = font, 6
				im := textBoxImage(t)
				im.ID, im.Y, im.Width, im.Height = "image", test.y, 40, 10
				doc, err := f.LayoutWithImages(strings.Repeat("正文Ab，\n", 9), s, []ImageAsset{im})
				if err != nil || len(doc.Pages) != 1 {
					t.Fatal("fixture did not fit one page", err)
				}
				assertWritingBlocks(t, doc.Pages[0], test.want)
				assertWritingReplay(t, doc)
			})
		}
	}
}

func TestWritingOrderIncludesTablesCaptionsAndOtherPages(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	top := textBoxImage(t)
	top.ID, top.Y, top.Width, top.Height = "top", 0, 40, 10
	bottom := top
	bottom.ID, bottom.Y = "bottom", 210
	table := sampleTable()
	table.Y = 60
	caption := sampleTextBox()
	caption.Page, caption.Y = 0, 120
	other := top
	other.ID, other.Page = "next-page", 1
	text := strings.Repeat("正文Ab，\n", 9) + "\f第二页正文"
	doc, err := f.LayoutWithObjects(text, s, []ImageAsset{bottom, other, top}, []TableAsset{caption, table})
	if err != nil || len(doc.Pages) != 2 {
		t.Fatal("fixture did not fit two pages", err)
	}
	assertWritingBlocks(t, doc.Pages[0], []string{"top", "body", "table", "body", "caption", "body", "bottom"})
	assertWritingBlocks(t, doc.Pages[1], []string{"next-page", "body"})
	assertWritingReplay(t, doc)
	reversed, err := f.LayoutWithObjects(text, s, []ImageAsset{top, other, bottom}, []TableAsset{table, caption})
	if err != nil || !reflect.DeepEqual(doc.Pages[0].Strokes, reversed.Pages[0].Strokes) {
		t.Fatal("insertion order affected writing", err)
	}
}

func TestWritingOrderUsesVisibleRotatedInkAndLeftToRightTies(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	left := textBoxImage(t)
	left.ID, left.X, left.Y, left.Width, left.Height = "left", 10, 60, 30, 10
	right := left
	right.ID, right.X = "right", 80
	doc, err := f.LayoutWithImages("正文", s, []ImageAsset{right, left})
	if err != nil {
		t.Fatal(err)
	}
	assertWritingBlocks(t, doc.Pages[0], []string{"body", "left", "right"})
	// The image's empty frame starts above the body, but its ink is below it.
	left.Y, left.Height = 0, 100
	doc, err = f.LayoutWithImages("正文", s, []ImageAsset{left})
	if err != nil {
		t.Fatal(err)
	}
	assertWritingBlocks(t, doc.Pages[0], []string{"body", "left"})
	left.X, left.Y, left.Width, left.Height, left.Rotation = 40, 45, 60, 10, 90
	right.Y = 35
	doc, err = f.LayoutWithImages("", s, []ImageAsset{right, left})
	if err != nil {
		t.Fatal(err)
	}
	assertWritingBlocks(t, doc.Pages[0], []string{"left", "right"})
}

func TestWritingOrderSurvivesErasureAndReload(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	im := textBoxImage(t)
	im.Y, im.Width, im.Height = 0, 40, 10
	text := "正文Ab，\n第二行"
	base, err := f.LayoutWithImages(text, s, []ImageAsset{im})
	if err != nil {
		t.Fatal(err)
	}
	// Split the top image's horizontal stroke into two pieces.
	e := PageErasure{ID: "cut", X: 25, Y: 0, Width: 5, Height: 10}
	doc, err := f.LayoutWithErasures(text, s, []ImageAsset{im}, nil, []PageErasure{e})
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Pages[0]
	if len(page.Strokes) <= len(base.Pages[0].Strokes) || !containsInk(page.Strokes, Point{15, 4}) || containsInk(page.Strokes, Point{27, 4}) {
		t.Fatal("erasure lost surviving image ink or restored removed ink")
	}
	if !reflect.DeepEqual(page.Strokes[len(page.Strokes)-len(page.TextStrokes):], page.TextStrokes) {
		t.Fatal("erasure put body before the top image or scrambled characters")
	}
	assertWritingReplay(t, doc)
	raw, _ := json.Marshal(doc.Erasures)
	var restored []PageErasure
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	reopened, err := f.LayoutWithErasures(text, s, []ImageAsset{im}, nil, restored)
	if err != nil || !reflect.DeepEqual(page.Strokes, reopened.Pages[0].Strokes) {
		t.Fatal("reloading materialized cuts changed writing order", err)
	}
}
