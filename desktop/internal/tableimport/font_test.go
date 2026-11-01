package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"strings"
	"testing"
)

func TestImportMeasuresSelectedFont(t *testing.T) {
	p := source()
	p.Layout.FontID = engine.FontDC
	e := fixture()
	e.Tables[0].Title = strings.Repeat("测量位置123", 8)
	plan, err := Build(font(t), p, e, Placement{})
	if err != nil {
		t.Fatal(err)
	}
	dc, _ := engine.BuiltinFont(engine.FontDC)
	o := plan.Objects[0]
	w, _ := dc.MeasureTextBoxText(o.Cells[0].Text, o.FontSize, p.Layout)
	if w+2*o.Padding > sum(o.ColumnWidths)+.001 || o.FontSize < p.Layout.FontSize*.7 {
		t.Fatal("import fitted using another font's measurements")
	}
	assertImportRendersAndExports(t, p, plan)
}
