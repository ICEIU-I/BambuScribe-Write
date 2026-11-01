package application

import (
	"bambuscribe/desktop/internal/engine"
	"testing"
)

func TestWritingRulePreviewsKeepTheCurrentExportDocument(t *testing.T) {
	a, _, _ := New(nil)
	doc, err := a.Preview("当前正文", engine.DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{`x_{i}^{2}`, `\frac{a}{b}`, `\sqrt{x^{2}+y^{2}}`, `\bar{U}`, `\[R=\frac{U}{I}\]`} {
		preview, err := a.PreviewWritingRule(text, engine.FontDC)
		if err != nil || len(preview.Pages[0].Strokes) == 0 {
			t.Fatal(err)
		}
	}
	if len(a.documentOrder) != 1 {
		t.Fatal("help modified the export cache")
	}
	if _, err := a.document(doc.ID); err != nil {
		t.Fatal("help evicted current document", err)
	}
}
