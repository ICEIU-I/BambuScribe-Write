package engine

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"strings"
	"testing"
)

// Draw to every edge so wrapping tests exercise the entire image envelope,
// while the separate margin test covers deliberately empty raster borders.
func wrappingImage(t *testing.T) ImageAsset {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for x := 0; x < 200; x++ {
		img.Set(x, 0, color.Black)
		img.Set(x, 99, color.Black)
	}
	for y := 0; y < 100; y++ {
		img.Set(0, y, color.Black)
		img.Set(199, y, color.Black)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return ImageAsset{ID: "outline", Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes()), TextRegions: []ImageTextRegion{}}
}

func TestBodyRemainsWholeBelowAnImage(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	im := wrappingImage(t)
	im.X, im.Y, im.Width, im.Height = 30, 0, 50, 40
	text := strings.Repeat("你好", 45)
	doc, err := f.LayoutWithImages(text, s, []ImageAsset{im})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, doc)
	page := doc.Pages[0]
	for i, c := range page.Cells {
		if c.Y < 41 {
			t.Fatal("body was split around an image", c)
		}
		if c.Start != i || c.End != i+1 {
			t.Fatal("wrapping duplicated or lost text", c)
		}
	}
	if page.Text != text || len(page.Cells) != len([]rune(text)) {
		t.Fatal("wrapping lost or reordered body text")
	}
	if _, err := ExportPage(doc, 0, testWriting()); err != nil {
		t.Fatal(err)
	}
}

func TestBodySkipsAllObjectGapsUntilAnUninterruptedRow(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	objects := []RegionBounds{{30, 0, 30, 20}, {90, 0, 30, 20}, {0, 30, 160, 20}}
	text := strings.Repeat("你", 80)
	doc, err := f.layoutText(text, s, map[int][]RegionBounds{0: objects})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range doc.Pages[0].Cells {
		if c.Y < 60 {
			t.Fatal("body entered an interrupted row", c)
		}
	}
	if doc.Pages[0].Cells[0].Y != 60 || doc.Pages[0].Text != text {
		t.Fatal("body did not resume immediately after the objects")
	}
}

func TestWholeLineAvoidancePreservesPunctuationUTF16IndentAndAlignment(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize, s.Indent = 6, 1
	objects := map[int][]RegionBounds{0: {{13.1, 0, 40, 30}}}
	for _, align := range []string{"left", "center", "right"} {
		s.Align = align
		text := "你，好🙂" + strings.Repeat("你好，", 15)
		plain, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := f.layoutText(text, s, objects)
		if err != nil {
			t.Fatal(err)
		}
		cells := append([]GlyphCell(nil), doc.Pages[0].Cells...)
		for i := range cells {
			cells[i].Y -= 45
		}
		if !reflect.DeepEqual(cells, plain.Pages[0].Cells) {
			t.Fatal("object avoidance changed line breaks, indent, alignment or UTF-16 offsets", align)
		}
		if cells[3].Start != 3 || cells[3].End != 5 || cells[4].Start != 5 {
			t.Fatal("offsets do not match textarea UTF-16")
		}
	}
}

func TestTallInkAndSelectionCellsBothAvoidObjects(t *testing.T) {
	f := &Font{Units: 1, Glyphs: map[rune]Glyph{'a': {Width: 1, Strokes: []Stroke{{{0, 0}, {1, 1.5}}}}}}
	s := DefaultLayout()
	s.FontSize = 6
	im := wrappingImage(t)
	im.X, im.Y, im.Width, im.Height = 0, 7, 30, 20
	doc, err := f.LayoutWithImages("aa", s, []ImageAsset{im})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, doc)
	if doc.Pages[0].Cells[0].X != 0 || doc.Pages[0].Cells[0].Y != 30 {
		t.Fatal("ink extending below its selection cell did not wrap")
	}
}

func TestExplicitBlankLinesKeepSavedSpacingAroundFullWidthObjects(t *testing.T) {
	s := DefaultLayout()
	s.FontSize = 6
	im := wrappingImage(t)
	im.X, im.Y, im.Width, im.Height = 0, 0, s.PaperWidth, 20
	doc, err := testFont(t).LayoutWithImages("\n\n正文", s, []ImageAsset{im})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Cells[0].Y != 30 || doc.Pages[0].Text != "\n\n正文" {
		t.Fatal("an object added rows on top of explicitly saved blank lines")
	}
	assertBodyExcludesObjects(t, doc)
}

func TestImageWhiteMarginsRetainExistingTextGeometry(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	im := textBoxImage(t)
	im.X, im.Y, im.Width, im.Height = 0, 0, s.PaperWidth, 60
	text := "原标题\n"
	plain, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := f.LayoutWithImages(text, s, []ImageAsset{im})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, doc)
	if !reflect.DeepEqual(doc.Pages[0].Cells, plain.Pages[0].Cells) || !reflect.DeepEqual(doc.Pages[0].TextStrokes, plain.Pages[0].TextStrokes) || len(doc.Pages) != 1 {
		t.Fatal("empty raster margins displaced previously non-overlapping text")
	}
}

func TestSingleHorizontalImageLineStillExcludesText(t *testing.T) {
	s := DefaultLayout()
	s.FontSize = 6
	im := textBoxImage(t)
	im.X, im.Y, im.Width, im.Height = 0, 0, 80, 15
	doc, err := testFont(t).LayoutWithImages("你好你好", s, []ImageAsset{im})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, doc)
	if doc.Pages[0].Cells[0].X != 0 || doc.Pages[0].Cells[0].Y != 15 {
		t.Fatal("a horizontal image line was treated as empty")
	}
}
