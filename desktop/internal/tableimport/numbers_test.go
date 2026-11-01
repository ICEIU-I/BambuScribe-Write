package tableimport

import (
	"encoding/json"
	"math"
	"testing"
)

func TestRecognizedThousandsHaveNoInternalSpaces(t *testing.T) {
	for input, want := range map[string]string{
		"1 000": "1000", "10 000": "10000", "1 000 000": "1000000",
		"I_{M} = 1\u202f000 mA；U = -12\u00a0345.67 V": "I_{M} = 1000 mA；U = -12345.67 V",
		"1\u2009000×10⁻¹¹ A":                          "1000×10⁻¹¹ A",
		"1 2 3":                                       "1 2 3", "1000 2000": "1000 2000", "1 23": "1 23",
		"1\t000\n1\n000": "1\t000\n1\n000", "10.00 mA": "10.00 mA",
	} {
		if got := normalizeNumbers(input); got != want {
			t.Errorf("%q became %q, want %q", input, got, want)
		}
	}
}

func TestImportedNumbersKeepPhysicalRowsAndFillTheirInkArea(t *testing.T) {
	p := source()
	e := Extraction{Tables: []Table{{Rows: 1, Columns: []float64{1}, Style: "grid", Cells: []Cell{{RowSpan: 1, ColSpan: 1, Text: "0.00"}}}}}
	plan, err := Build(font(t), p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	a, rendered := plan.Objects[0], plan.Preview[0]
	if rendered.Cells[0].FontSize > p.Layout.FontSize+1e-8 || a.RowHeights[0] > p.Layout.FontSize+2*a.Padding+1e-8 {
		t.Fatal("import enlarged the digits or retained an oversized row")
	}
	top, bottom := math.Inf(1), math.Inf(-1)
	for _, stroke := range rendered.Cells[0].Strokes {
		for _, point := range stroke {
			top, bottom = min(top, point.Y), max(bottom, point.Y)
		}
	}
	if math.Abs(bottom-top-(a.RowHeights[0]-2*a.Padding)) > 1e-7 {
		t.Fatal("newly imported digits left unnecessary vertical whitespace", top, bottom)
	}
}

func TestNumbersNormalizedBeforeReviewAndDirectImport(t *testing.T) {
	e := fixture()
	e.Tables[0].Title = "表1 1 000 mA"
	e.Tables[0].Notes = []string{"I_{M} = 1 000 mA"}
	e.Tables[0].Cells[0].Text = "1 000"
	raw, _ := json.Marshal(e)
	decoded, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Tables[0].Title != "表1 1000 mA" || decoded.Tables[0].Notes[0] != "I_{M}=1000 mA" || decoded.Tables[0].Cells[0].Text != "1000" {
		t.Fatal("recognition review still exposes thousands separators", decoded)
	}
	// Build also accepts structured extractions without passing through Decode.
	plan, err := Build(font(t), source(), e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"表1 1000 mA", "I_{M}=1000 mA", "1000"} {
		if plan.Objects[i].Cells[0].Text != want {
			t.Fatal("direct import retained thousands separators", plan.Objects[i])
		}
	}
	if e.Tables[0].Cells[0].Text != "1 000" {
		t.Fatal("planning modified the caller's extraction")
	}
}
