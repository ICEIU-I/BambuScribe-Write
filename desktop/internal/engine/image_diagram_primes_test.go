package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"encoding/json"
	"reflect"
	"testing"
)

func TestDiagramPrimeLabelsMatchUnicodeInk(t *testing.T) {
	f := testFont(t)
	for _, tc := range []struct{ source, want string }{
		{`p^{\prime}`, "p′"},
		{`p^\prime`, "p′"},
		{`p\prime`, "p′"},
		{`p^{ \prime }`, "p′"},
		{`p^{\prime\prime}`, "p″"},
		{`p^{\prime \prime}`, "p″"},
		{"p^{′}", "p′"},
		{"p^{″}", "p″"},
		{`p_{1}^{\prime}`, "p_{1}′"},
		{`p^{\prime}+x^{2}+E_{H}`, "p′+x^{2}+E_{H}"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			scene := &diagram.Scene{
				Width: 200, Height: 100,
				Strokes: []diagram.Stroke{{ID: "ray", Points: []diagram.Point{{X: 10, Y: 20}, {X: 180, Y: 20}}}},
				Labels:  []diagram.Label{{Text: tc.source, X: 30, Y: 40, Width: 140, Height: 30}},
			}
			if err := f.ValidateDiagram(scene); err != nil {
				t.Fatal("model prime notation blocked the image:", err)
			}
			asset := ImageAsset{Width: 100, Height: 50, Diagram: scene}
			got, err := diagramImageStrokes(f, asset, DefaultLayout())
			if err != nil {
				t.Fatal(err)
			}
			if scene.Labels[0].Text != tc.source {
				t.Fatal("rendering changed the saved model result")
			}
			wantScene := *scene
			wantScene.Labels = append([]diagram.Label(nil), scene.Labels...)
			wantScene.Labels[0].Text = tc.want
			asset.Diagram = &wantScene
			want, err := diagramImageStrokes(f, asset, DefaultLayout())
			if err != nil || len(want.Strokes) <= 1 {
				t.Fatal("Unicode reference did not render:", err)
			}
			if !reflect.DeepEqual(got.Strokes, want.Strokes) {
				t.Fatal("prime size/baseline, scripts or diagram geometry differ from Unicode")
			}
		})
	}
}

func TestDiagramPrimeSurvivesProjectReloadAndGCodeExport(t *testing.T) {
	f := testFont(t)
	asset := ImageAsset{
		ID: "optics", Data: testImageData(t), X: 10, Y: 10, Width: 100, Height: 50,
		Diagram: &diagram.Scene{Width: 200, Height: 100, Labels: []diagram.Label{{Text: `p^{\prime}`, X: 30, Y: 40, Width: 20, Height: 15}}},
	}
	raw, err := json.Marshal(asset)
	if err != nil {
		t.Fatal(err)
	}
	var restored ImageAsset
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	doc, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{restored})
	if err != nil {
		t.Fatal("saved prime label failed to lay out:", err)
	}
	if len(doc.Pages) != 1 || len(doc.Pages[0].Strokes) == 0 || len(doc.Missing) != 0 {
		t.Fatal("prime label lost its strokes or produced missing glyphs")
	}
	gcode, err := ExportPage(doc, 0, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	restored.Diagram.Labels[0].Text = "p′"
	want, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{restored})
	if err != nil {
		t.Fatal(err)
	}
	wantGCode, err := ExportPage(want, 0, testWriting())
	if err != nil || string(gcode) != string(wantGCode) {
		t.Fatal("export differs from the Unicode prime label:", err)
	}
}

func TestDiagramPrimeCompatibilityStillRejectsUnsupportedNotation(t *testing.T) {
	f := testFont(t)
	for _, text := range []string{`p\primed`, `p\primefoo`, `p^{\primed}`, `p^{\primefoo}`, `p^{\sqrt{2}}`, `\sqrt{p^{\prime}}`, `p^{2\prime}`, `p^{\prime`, `^{\prime}`, `p_^{\prime}`} {
		scene := &diagram.Scene{Width: 200, Height: 100, Labels: []diagram.Label{{Text: text, X: 30, Y: 40, Width: 140, Height: 30}}}
		if err := f.ValidateDiagram(scene); err == nil {
			t.Errorf("unsupported %q silently accepted", text)
		}
	}
}
