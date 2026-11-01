package engine

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDeletingTextPreservesImagePages(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.LineMode = "random"
	asset := textBoxImage(t)
	asset.Page = 2
	asset.TextRegions = []ImageTextRegion{{ID: "label", Text: "W/m²", X: .1, Y: .1, Width: .3, Height: .2, Mode: "handwriting", Source: "manual"}}
	assets := []ImageAsset{asset}
	original := strings.Repeat("你好\n", 40)
	before, err := f.LayoutWithImages(original, s, assets)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Pages) < 3 {
		t.Fatal("fixture must start with multiple text pages")
	}
	for _, text := range []string{"", "继续输入", original, ""} {
		doc, err := f.LayoutWithImages(text, s, assets)
		if err != nil {
			t.Fatalf("deleting/retyping/restoring text must keep image pages: %v", err)
		}
		if !reflect.DeepEqual(doc.Pages[2].Images, before.Pages[2].Images) {
			t.Fatal("text editing changed the image or its handwritten text box")
		}
		if text == original {
			if !reflect.DeepEqual(doc, before) {
				t.Fatal("undo did not restore the original layout")
			}
			continue
		}
		if len(doc.Pages) != 3 {
			t.Fatal("page count must include the last image page")
		}
		textOnly, err := f.Layout(text, s)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(doc.Pages[0], textOnly.Pages[0]) {
			t.Fatal("image pages changed the remaining text or its caret")
		}
		for i, page := range doc.Pages[1:] {
			count, height := s.rows(i + 1)
			if page.Number != i+2 || page.LineCount != count || page.LineHeight != height {
				t.Fatal("image-only pages lost their numbering or ruled grid")
			}
			if len(page.TextStrokes) != 0 || len(page.Cells) != 0 || len(page.Carets) != 0 {
				t.Fatal("image-only pages retained deleted text or duplicate caret positions")
			}
			raw, err := json.Marshal(page)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"strokes", "textStrokes", "cells", "carets", "missing"} {
				if bytes.Contains(raw, []byte(`"`+field+`":null`)) {
					t.Fatalf("%s must be an array for the frontend", field)
				}
			}
		}
		if text == "" {
			data, err := ExportAll(doc, testWriting())
			if err != nil {
				t.Fatal(err)
			}
			archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Fatal(err)
			}
			if len(archive.File) != 1 || archive.File[0].Name != "page-003.gcode" {
				t.Fatal("export must preserve the image page and skip empty pages")
			}
		}
	}
	cleared, err := f.LayoutWithImages("", s, nil)
	if err != nil || len(cleared.Pages) != 1 || len(cleared.Pages[0].Carets) != 1 {
		t.Fatal("removing all content must leave one editable blank page")
	}
}

func TestImagePagesRejectInvalidIndices(t *testing.T) {
	f := testFont(t)
	asset := textBoxImage(t)
	for _, page := range []int{-1, 20001, int(^uint(0) >> 1)} {
		asset.Page = page
		if _, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset}); err == nil {
			t.Fatalf("invalid image page %d accepted", page)
		}
	}
}
