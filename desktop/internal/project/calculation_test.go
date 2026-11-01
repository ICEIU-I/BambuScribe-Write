package project

import (
	"bambuscribe/desktop/internal/calculation"
	"bambuscribe/desktop/internal/engine"
	"os"
	"reflect"
	"testing"
)

func TestCalculationProjectRetainsRulesAndFormulaInk(t *testing.T) {
	raw, err := os.ReadFile("../calculation/testdata/hall.json")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := calculation.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	result, err := plan.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	settings := engine.DefaultLayout()
	settings.FontSize = 5
	settings.FractionScale = 75
	writing := engine.DefaultWriting()
	writing.OriginMode = "auto"
	down, up := 3.4, 7.0
	writing.PenDown = &down
	writing.PenUp = &up
	p := Project{Version: Version, Text: result.Text, Layout: settings, Writing: writing, Calculations: []calculation.Record{{ID: "hall", Text: result.Text, Plan: plan}}}
	encoded, err := Encode(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.Calculations, p.Calculations) || restored.Layout.FractionScale != 75 {
		t.Fatal("editable calculation rules lost")
	}
	f, err := engine.EmbeddedFont()
	if err != nil {
		t.Fatal(err)
	}
	a, err := f.Layout(p.Text, p.Layout)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.Layout(restored.Text, restored.Layout)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Missing) > 0 || !reflect.DeepEqual(a.Pages, b.Pages) {
		t.Fatal("formula preview changed")
	}
	for i := range a.Pages {
		ga, err := engine.GenerateGCode(a.Pages[i], a.Settings, writing)
		if err != nil {
			t.Fatal(err)
		}
		gb, err := engine.GenerateGCode(b.Pages[i], b.Settings, writing)
		if err != nil || ga != gb {
			t.Fatal("export changed after reopen", err)
		}
	}
}
