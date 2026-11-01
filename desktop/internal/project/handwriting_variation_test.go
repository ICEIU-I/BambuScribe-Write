package project

import (
	"bambuscribe/desktop/internal/engine"
	"reflect"
	"testing"
)

func TestHandwritingVariationSurvivesProjectRoundTrip(t *testing.T) {
	p := Project{Text: "霍尔效应888\n\\(U=\\frac{88}{88}\\)", Layout: engine.DefaultLayout(), Writing: engine.DefaultWriting()}
	p.Layout.FontSize = 6
	p.Layout.HandwritingVariation, p.Layout.HandwritingSeed = 45, 4294967295
	f, _ := engine.EmbeddedFont()
	a, err := f.Layout(p.Text, p.Layout)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.Layout(restored.Text, restored.Layout)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("saved handwriting changed on reopen")
	}
	for i := range a.Pages {
		left, e1 := engine.GenerateGCode(a.Pages[i], a.Settings, p.Writing)
		right, e2 := engine.GenerateGCode(b.Pages[i], b.Settings, restored.Writing)
		if e1 != nil || e2 != nil || left != right {
			t.Fatal("reopened export changed", e1, e2)
		}
	}
}
