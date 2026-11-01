package application

import (
	"bambuscribe/desktop/internal/engine"
	"strings"
	"testing"
)

func uploadDocument() *engine.Document {
	return &engine.Document{ID: "batch", Settings: engine.DefaultLayout(), Pages: []engine.Page{
		{Number: 1, Strokes: []engine.Stroke{{{X: 30, Y: 30}, {X: 32, Y: 32}}}},
		{Number: 2},
		{Number: 3, Strokes: []engine.Stroke{{{X: 35, Y: 35}, {X: 36, Y: 36}}}},
	}}
}

func TestUploadPreparesPlainGCodeAndPreservesPageNumbers(t *testing.T) {
	pages, err := uploadPages(uploadDocument(), engine.DefaultWriting())
	if err != nil || len(pages) != 2 || pages[0].Page != 1 || pages[1].Page != 3 {
		t.Fatalf("pages=%+v err=%v", pages, err)
	}
	for i, page := range pages {
		if !strings.HasPrefix(string(page.Data), "; A1 handwriting / page ") {
			t.Fatal("upload must contain plain G-code, not a sliced archive")
		}
		want := "/ page 1"
		if i == 1 {
			want = "/ page 3"
		}
		if !strings.Contains(string(page.Data), want) {
			t.Fatal("wrong page in uploaded G-code")
		}
	}
}

func TestUploadValidatesLaterPagesBeforePreparingAnyBatch(t *testing.T) {
	for _, problem := range []string{"missing", "bounds"} {
		d := uploadDocument()
		if problem == "missing" {
			d.Pages[2].Missing = []engine.MissingGlyph{{Char: "🙂"}}
		} else {
			d.Pages[2].Strokes[0][1].X = 600
		}
		pages, err := uploadPages(d, engine.DefaultWriting())
		if err == nil || pages != nil || !strings.Contains(err.Error(), "第 3 页") {
			t.Fatalf("pages=%+v err=%v", pages, err)
		}
	}
	if _, err := uploadPages(&engine.Document{}, engine.DefaultWriting()); err == nil {
		t.Fatal("empty document accepted")
	}
}

func TestA4BatchUsesFrameTravelAndPreflightsEveryPage(t *testing.T) {
	d := uploadDocument()
	d.Settings.PaperWidth, d.Settings.PaperHeight = 210, 297
	d.Settings.MarginLeft, d.Settings.MarginRight = 25.4, 25.4
	d.Settings.MarginTop, d.Settings.MarginBottom = 25.4, 25.4
	d.Pages[0].Strokes = []engine.Stroke{{{X: 25.4, Y: 25.4}, {X: 184.6, Y: 271.6}}}
	d.Pages[2].Strokes = []engine.Stroke{{{X: 25.4, Y: 25.4}, {X: 184.6, Y: 271.6}}}
	w := engine.DefaultWriting()
	pages, err := uploadPages(d, w)
	if err != nil || len(pages) != 2 || pages[0].Page != 1 || pages[1].Page != 3 {
		t.Fatalf("valid A4 batch blocked: pages=%+v err=%v", pages, err)
	}
	d.Pages[2].Strokes[0][1].Y = 282
	pages, err = uploadPages(d, w)
	if err == nil || pages != nil || !strings.Contains(err.Error(), "第 3 页") {
		t.Fatalf("later-page travel must block the entire batch: pages=%+v err=%v", pages, err)
	}
}

func TestSpecificUploadOnlyValidatesTheSelectedPage(t *testing.T) {
	d := uploadDocument()
	d.Pages[2].Missing = []engine.MissingGlyph{{Char: "🙂"}}
	pages, err := uploadPages(d, engine.DefaultWriting(), 0)
	if err != nil || len(pages) != 1 || pages[0].Page != 1 {
		t.Fatalf("pages=%+v err=%v", pages, err)
	}
	for _, indices := range [][]int{{-1}, {3}, {0, 0}, {2, 0}, {1}} {
		if _, err := uploadPages(d, engine.DefaultWriting(), indices...); err == nil {
			t.Fatalf("invalid indices accepted: %v", indices)
		}
	}
}
