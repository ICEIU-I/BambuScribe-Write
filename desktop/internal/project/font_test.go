package project

import (
	"bambuscribe/desktop/internal/engine"
	"reflect"
	"testing"
)

func TestFontChoiceRoundTrip(t *testing.T) {
	for _, id := range []string{"", engine.FontHandwriting, engine.FontDC} {
		p := Project{Version: Version, Text: "实验霍尔效应", Layout: engine.DefaultLayout(), Writing: engine.DefaultWriting(), Images: []engine.ImageAsset{}}
		p.Layout.FontID = id
		b, err := Encode(p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Decode(b)
		if err != nil || !reflect.DeepEqual(got.Layout, p.Layout) {
			t.Fatal("project changed font or layout", err)
		}
		base, _ := engine.EmbeddedFont()
		d, err := base.Layout(got.Text, got.Layout)
		if err != nil || len(d.Missing) != 0 || (id == engine.FontDC && d.FontName != "DC体") {
			t.Fatal("reopened project used wrong font", err)
		}
	}
}
