package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"math"
	"strings"
	"testing"
)

func TestImportMeasuresSelectedFont(t *testing.T) {
	p := source()
	p.Layout.FontID = engine.FontDC
	e := fixture()
	e.Tables[0].Title = strings.Repeat("1234567890", 8)
	plan, err := Build(font(t), p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	dc, _ := engine.BuiltinFont(engine.FontDC)
	o := plan.Objects[0]
	w, _ := dc.MeasureTableText(o.Cells[0].Text, o.FontSize)
	if math.Abs(w+2*o.Padding-sum(o.ColumnWidths)) > .001 {
		t.Fatal("import fitted using another font's measurements")
	}
	assertImportRendersAndExports(t, p, plan)
}
