package engine

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

// Check public preview geometry, not the planner's internal occupancy map.
func assertBodyExcludesObjects(t *testing.T, doc *Document) {
	t.Helper()
	for _, page := range doc.Pages {
		objects := []RegionBounds{}
		for _, im := range page.Images {
			a := im.Rotation * math.Pi / 180
			w := math.Abs(im.Width*math.Cos(a)) + math.Abs(im.Height*math.Sin(a))
			h := math.Abs(im.Width*math.Sin(a)) + math.Abs(im.Height*math.Cos(a))
			objects = append(objects, RegionBounds{im.X + (im.Width-w)/2, im.Y + (im.Height-h)/2, w, h})
		}
		for _, table := range page.Tables {
			objects = append(objects, RegionBounds{table.X, table.Y, table.Width, table.Height})
		}
		for _, a := range objects {
			if a.X < -1e-8 || a.Y < -1e-8 || a.X+a.Width > doc.Settings.PaperWidth+1e-8 || a.Y+a.Height > doc.Settings.PaperHeight+1e-8 {
				t.Fatalf("page %d object outside paper: %+v", page.Number, a)
			}

		}
		// Text clears image ink, including rotated strokes, rather than the
		// empty margins inside its draggable frame.
		bodyObjects := []RegionBounds{}
		for _, im := range page.Images {
			bounds, ok := imageInkBounds(im.Strokes)
			if ok {
				bodyObjects = append(bodyObjects, bounds)
			}
		}
		for _, table := range page.Tables {
			bodyObjects = append(bodyObjects, RegionBounds{table.X, table.Y, table.Width, table.Height})
		}
		for _, a := range bodyObjects {
			for _, c := range page.Cells {
				if tableOverlaps(c.X, c.Y, c.Width, c.Height, a.X, a.Y, a.Width, a.Height) {
					t.Fatalf("page %d text cell entered an object: %+v / %+v", page.Number, c, a)
				}
			}
			for _, stroke := range page.TextStrokes {
				for _, p := range stroke {
					if p.X >= a.X && p.X <= a.X+a.Width && p.Y >= a.Y && p.Y <= a.Y+a.Height {
						t.Fatalf("page %d text ink entered an object: %+v / %+v", page.Number, p, a)
					}
				}
			}
		}
	}
}

func TestOverlappingMixedObjectsKeepAnchorsAndExcludeBody(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	image := textBoxImage(t)
	image.X, image.Y, image.Width, image.Height, image.Rotation = 20, 20, 65, 30, 25
	otherImage := image
	otherImage.ID = "second-image"
	table := sampleTable()
	table.X, table.Y = 20, 20
	caption := sampleTextBox()
	caption.Page, caption.X, caption.Y = 0, 20, 20
	text := strings.Repeat("正文文字不会丢失。\n", 65)
	doc, err := f.LayoutWithObjects(text, s, []ImageAsset{image, otherImage}, []TableAsset{table, caption})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, doc)
	var body string
	indices := map[int]bool{}
	for _, p := range doc.Pages {
		body += p.Text
		for _, c := range p.Cells {
			if indices[c.Start] {
				t.Fatal("duplicate text cell", c.Start)
			}
			indices[c.Start] = true
		}
		if len(p.Warnings) > 0 || len(p.Issues) > 0 {
			t.Fatal(p.Warnings, p.Issues)
		}
	}
	if body != text {
		t.Fatal("avoidance dropped or reordered body text")
	}
	if len(doc.ObjectLayout) != 4 {
		t.Fatal("objects were dropped")
	}
	for _, position := range doc.ObjectLayout {
		if position.Page != 0 || position.X != 20 || position.Y != 20 {
			t.Fatal("overlapping object was moved", position)
		}
	}
	if _, err := ExportAll(doc, testWriting()); err != nil {
		t.Fatal(err)
	}
	// Input anchors remain editable, and reversing array order cannot alter the
	// resolved reading order or placement.
	reversed, err := f.LayoutWithObjects(text, s, []ImageAsset{otherImage, image}, []TableAsset{caption, table})
	if err != nil || !reflect.DeepEqual(doc.ObjectLayout, reversed.ObjectLayout) {
		t.Fatal("unstable placement", err)
	}
}

func TestFullPageImageMovesOnlyBodyToNextPage(t *testing.T) {
	s := DefaultLayout()
	s.FontSize = 6
	im := wrappingImage(t)
	im.X, im.Y, im.Width, im.Height = 0, 0, s.PaperWidth, s.PaperHeight
	table := sampleTable()
	table.X, table.Y = 10, 10
	doc, err := testFont(t).LayoutWithObjects("开头\n正文\n", s, []ImageAsset{im}, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) < 2 || doc.Pages[0].Text != "" || len(doc.Pages[0].Tables) != 1 {
		t.Fatal("full-page image did not preserve the overlapping table and paginate body")
	}
	assertBodyExcludesObjects(t, doc)
	if len(doc.Pages[0].Carets) != 0 || doc.Pages[1].Carets[0].Index != 0 {
		t.Fatal("caret stayed inside the image or lost its offset")
	}
}

func TestGrowingTableReservesMeasuredHeightAndRestoresOnUndo(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.FontSize = 6
	s.LineMode = "random"
	table := sampleTable()
	table.X, table.Y = 15, 20
	box := sampleTextBox()
	box.Page, box.X, box.Y = 0, 15, 41
	text := strings.Repeat("实验正文\n", 45)
	before, err := f.LayoutWithObjects(text, s, nil, []TableAsset{table, box})
	if err != nil {
		t.Fatal(err)
	}
	grown := table
	grown.Cells = append([]TableCell(nil), table.Cells...)
	grown.RowHeights = []float64{90, 10}
	after, err := f.LayoutWithObjects(text, s, nil, []TableAsset{grown, box})
	if err != nil {
		t.Fatal(err)
	}
	assertBodyExcludesObjects(t, after)
	for _, position := range after.ObjectLayout {
		if position.ID == box.ID && (position.Y != box.Y || position.Page != box.Page) {
			t.Fatal("table growth moved the overlapping caption")
		}
	}
	if reflect.DeepEqual(before.Pages[0].Cells, after.Pages[0].Cells) {
		t.Fatal("body did not move below the grown table")
	}
	if !reflect.DeepEqual(table.RowHeights, []float64{10, 10}) || box.Y != 41 {
		t.Fatal("planner mutated the saved minimum heights/anchors")
	}
	undone, err := f.LayoutWithObjects(text, s, nil, []TableAsset{table, box})
	if err != nil || !reflect.DeepEqual(before, undone) {
		t.Fatal("undo did not restore layout", err)
	}
	plain, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	removed, err := f.LayoutWithObjects(text, s, nil, nil)
	if err != nil || !reflect.DeepEqual(plain.Pages, removed.Pages) {
		t.Fatal("removing objects did not restore text flow", err)
	}
}

func TestObjectsTooLargeForAnyPageAreRejected(t *testing.T) {
	table := sampleTable()
	table.ColumnWidths = []float64{100, 100}
	if _, err := testFont(t).LayoutWithObjects("正文", DefaultLayout(), nil, []TableAsset{table}); err == nil || !strings.Contains(err.Error(), "整页") {
		t.Fatal("oversized table accepted", err)
	}
	image := textBoxImage(t)
	image.Width, image.Height, image.Rotation = 160, 230, 45
	if _, err := testFont(t).LayoutWithImages("正文", DefaultLayout(), []ImageAsset{image}); err == nil {
		t.Fatal("oversized rotated image accepted")
	}
}

func TestExplicitPlacementCanUseTheBottomMargin(t *testing.T) {
	s := DefaultLayout()
	s.MarginTop, s.MarginBottom = 20, 25
	s.FontSize = 6
	table := sampleTable()
	table.Y = 200
	doc, err := testFont(t).LayoutWithObjects("正文", s, nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 || len(doc.Pages[0].Tables) != 1 || doc.Pages[0].Tables[0].Y != table.Y {
		t.Fatal("explicit object anchor could not use the bottom margin")
	}
	assertBodyExcludesObjects(t, doc)
	if _, err := ExportPage(doc, 0, testWriting()); err != nil {
		t.Fatal("margin object could not be exported", err)
	}
}

func TestOverlappingMarginObjectsKeepTheirAnchors(t *testing.T) {
	s := DefaultLayout()
	s.MarginTop, s.MarginBottom = 20, 25
	s.FontSize = 6
	table := sampleTable()
	table.Y = 195
	other := table
	other.ID, other.Y = "other", 200
	doc, err := testFont(t).LayoutWithObjects("正文", s, nil, []TableAsset{table, other})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 || len(doc.Pages[0].Tables) != 2 || doc.Pages[0].Tables[1].Y != other.Y {
		t.Fatal("overlapping margin objects were moved")
	}
	assertBodyExcludesObjects(t, doc)
}

func TestMarginObjectPastPaperEdgeStillMovesToNextPage(t *testing.T) {
	s := DefaultLayout()
	s.MarginTop, s.MarginBottom = 20, 25
	table := sampleTable()
	table.Y = s.PaperHeight - 5
	doc, err := testFont(t).LayoutWithObjects("", s, nil, []TableAsset{table})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 2 || len(doc.Pages[0].Tables) != 0 || doc.Pages[1].Tables[0].Y != s.MarginTop {
		t.Fatal("paper edge overflow was not moved to the next page")
	}
	assertBodyExcludesObjects(t, doc)
}
