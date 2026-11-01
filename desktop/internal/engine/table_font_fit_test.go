package engine

import (
	"reflect"
	"testing"
)

func assertMaximumTableFont(t *testing.T, f *Font, table TableAsset, render TablePlacement) {
	t.Helper()
	for _, cell := range render.Cells {
		for _, stroke := range cell.Strokes {
			for _, p := range stroke {
				if p.X < cell.X+render.Padding-1e-7 || p.X > cell.X+cell.Width-render.Padding+1e-7 || p.Y < cell.Y+render.Padding-1e-7 || p.Y > cell.Y+cell.Height-render.Padding+1e-7 {
					t.Fatal("maximum font clipped ink", cell.Text, p)
				}
			}
		}
		plan := f.objectText(table, cell.Text, cell.FontSize+.001, 1e9, DefaultLayout())
		width, height := 0.0, plan.height
		for _, w := range plan.widths {
			width = max(width, w)
		}
		limited := width > cell.Width-2*render.Padding+1e-8 || height > cell.Height-2*render.Padding+1e-8
		if cell.Text != "" && !limited && cell.FontSize < render.MaxFontSize-1e-7 {
			t.Fatal("cell could use a larger font", cell.Text, cell.FontSize)
		}
	}
}

func TestTableFontGrowsAndShrinksToMaximumIncludingLegacySmallFonts(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.FontSize = 1
	table.Cells[0].Text = "U̅_{H}/mV"
	table.Cells[1].Text = "+I_{M},+I_{S}"
	initial := f.renderTable(table, DefaultLayout())
	if initial.FontSize <= table.FontSize {
		t.Fatal("legacy small font was retained")
	}
	assertMaximumTableFont(t, f, table, initial)
	for _, factor := range []float64{.5, 1.5, 2} {
		scaled := table
		scaled.ColumnWidths = []float64{30 * factor, 30 * factor}
		scaled.RowHeights = []float64{10 * factor, 10 * factor}
		scaled.Padding *= factor
		render := f.renderTable(scaled, DefaultLayout())
		if render.FontSize > render.MaxFontSize+1e-7 || render.Width > 60*factor+1e-7 {
			t.Fatal("font did not scale with cells", factor, render.FontSize, initial.FontSize)
		}
		assertMaximumTableFont(t, f, scaled, render)
	}
	table.FontSize = 40
	largeSaved := f.renderTable(table, DefaultLayout())
	if !reflect.DeepEqual(initial, largeSaved) {
		t.Fatal("saved font still overrides automatic fitting")
	}
}

func TestMaximumTableFontUsesMergedCellsLinesPaddingAndIgnoresEmptyCells(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.RowHeights = []float64{6, 6}
	table.Cells = []TableCell{
		{Row: 0, Col: 0, RowSpan: 2, ColSpan: 1, Text: "U_{AK}\n平均值"},
		{Row: 0, Col: 1, RowSpan: 1, ColSpan: 1},
		{Row: 1, Col: 1, RowSpan: 1, ColSpan: 1},
	}
	render := f.renderTable(table, DefaultLayout())
	assertMaximumTableFont(t, f, table, render)
	if !reflect.DeepEqual(render.RowHeights, table.RowHeights) || len(render.Cells[1].Strokes) != 0 || len(render.Cells[2].Strokes) != 0 {
		t.Fatal("fit changed merged geometry or filled a blank measurement cell")
	}
	before := render.FontSize
	table.Cells[0].Text = "U_{AK}"
	render = f.renderTable(table, DefaultLayout())
	if render.FontSize <= before {
		t.Fatal("shorter text did not reclaim available size")
	}
	assertMaximumTableFont(t, f, table, render)
	table.Padding = 2
	padded := f.renderTable(table, DefaultLayout())
	if padded.FontSize >= render.FontSize {
		t.Fatal("padding did not reduce available size")
	}
	assertMaximumTableFont(t, f, table, padded)
}

func TestMaximumTableFontKeepsMinimumFontErrors(t *testing.T) {
	f := testFont(t)
	table := sampleTable()
	table.ColumnWidths[0] = 3
	table.Cells[0].Text = "无法容纳的长文字"
	render := f.renderTable(table, DefaultLayout())
	if render.FontSize != 1 || len(render.Issues) == 0 {
		t.Fatal("text too long at minimum font was silently clipped")
	}
}

func TestMaximumTableFontFollowsSelectedFont(t *testing.T) {
	base := testFont(t)
	for _, id := range []string{FontHandwriting, FontDC} {
		t.Run(id, func(t *testing.T) {
			settings := DefaultLayout()
			settings.FontID = id
			table := sampleTable()
			table.FontSize = 1
			table.Cells[0].Text = "U̅_{H}/mV"
			table.Cells[1].Text = "实验结果"
			doc, err := base.LayoutWithObjects("正文", settings, nil, []TableAsset{table})
			if err != nil {
				t.Fatal(err)
			}
			selected, err := BuiltinFont(id)
			if err != nil {
				t.Fatal(err)
			}
			assertMaximumTableFont(t, selected, table, doc.Pages[0].Tables[0])
		})
	}
}
