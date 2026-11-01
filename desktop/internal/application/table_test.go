package application

import (
	"bambuscribe/desktop/internal/engine"
	"strings"
	"testing"
)

func TestTableIssuesBlockEntireUploadBeforeAnyTransfer(t *testing.T) {
	for _, visible := range []bool{true, false} {
		doc := uploadDocument()
		doc.Pages[2].Issues = []string{"表格超出纸张，请调整尺寸"}
		if !visible {
			doc.Pages[2].Strokes = nil
		}
		pages, err := uploadPages(doc, engine.DefaultWriting())
		if err == nil || pages != nil || !strings.Contains(err.Error(), "表格超出纸张") {
			t.Fatal("bad later table was transferred or skipped", pages, err)
		}
		valid, err := uploadPages(doc, engine.DefaultWriting(), 0)
		if err != nil || len(valid) != 1 {
			t.Fatal("bad unrelated table blocked a valid specific page", err)
		}
	}
}
