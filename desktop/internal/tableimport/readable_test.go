package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAnswerBlanksSurviveRecognitionAndPlanning(t *testing.T) {
	for _, input := range []string{"d̅_{1}=        mm", "d̅=√(d̅_{1}d̅_{2})=        mm", "x=        ", "测量值：        毫米", "x=(        )"} {
		if got := normalizeText(input); got != input {
			t.Errorf("writable blank %q became %q", input, got)
		}
		e := fixture()
		e.Tables[0].Cells[0].Text = input
		raw, _ := json.Marshal(e)
		decoded, err := Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		p := source()
		plan, err := Build(font(t), p, decoded, Placement{})
		if err != nil {
			t.Fatal(err)
		}
		if strings.ReplaceAll(plan.Objects[len(plan.Objects)-1].Cells[0].Text, "\n", "") != input {
			t.Fatal("planning lost blank", plan.Objects)
		}
		assertImportRendersAndExports(t, p, plan)
	}
	for input, want := range map[string]string{"I =    10.00 mA": "I=10.00 mA", "X =    Y": "X=Y", "Y =   ____ mm": "Y=____ mm", "Hall    effect": "Hall effect"} {
		if got := normalizeText(input); got != want {
			t.Errorf("%q became %q", input, got)
		}
	}
}

func TestReadableImportWrapsAndKeepsCaptionOnOneLine(t *testing.T) {
	for _, id := range []string{engine.FontHandwriting, engine.FontDC} {
		p := source()
		p.Layout.FontID = id
		e := fixture()
		e.Tables[0].Title = "表2 Δs=0.00 mm, Δe=35.00 mm"
		e.Tables[0].Cells[0].Text = strings.Repeat("狭缝面相对座中心偏移", 4) + "Δs/mm"
		e.Tables[0].Cells[1].Text = "d̅=√(d̅_{1}d̅_{2})=        mm"
		plan, err := Build(font(t), p, e, Placement{})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(plan.Objects[0].Cells[0].Text, "\n") {
			t.Fatal("one-line caption split")
		}
		table := plan.Objects[len(plan.Objects)-1]
		if !strings.Contains(table.Cells[0].Text, "\n") || table.Cells[0].RowSpan != 2 || table.Cells[1].ColSpan != 2 {
			t.Fatal("wrapping lost merged topology")
		}
		for _, preview := range plan.Preview {
			for _, cell := range preview.Cells {
				if strings.TrimSpace(cell.Text) != "" && cell.FontSize+1e-7 < p.Layout.FontSize*.7 {
					t.Fatalf("unreadable cell %.3f below %.3f", cell.FontSize, p.Layout.FontSize*.7)
				}
			}
		}
		assertImportRendersAndExports(t, p, plan)
	}
}

func TestImportWidensBeforeWrappingAndAccountsForCaptionSpacing(t *testing.T) {
	p := source()
	p.Layout.LetterSpacing = 140
	p.Layout.LetterGap = .7
	e := fixture()
	e.Tables[0].Title = "表2 Δs=0.00 mm, Δe=35.00 mm"
	plan, err := Build(font(t), p, e, Placement{Positions: []Position{{Page: 1, X: 0, Y: 0, Width: 45}}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Groups[0].Width <= 45 {
		t.Fatal("did not widen table before reducing readability")
	}
	if strings.Contains(plan.Objects[0].Cells[0].Text, "\n") {
		t.Fatal("caption wrapped despite available width")
	}
	for _, preview := range plan.Preview {
		for _, cell := range preview.Cells {
			if cell.Text != "" && cell.FontSize+1e-7 < p.Layout.FontSize*.7 {
				t.Fatal("caption spacing shrank font below floor", cell.FontSize)
			}
		}
	}
	assertImportRendersAndExports(t, p, plan)
}

func TestLensAndSlitSourceTablesKeepReadableFormulas(t *testing.T) {
	raw, err := os.ReadFile("testdata/lens-and-slit.json")
	if err != nil {
		t.Fatal(err)
	}
	e, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{engine.FontHandwriting, engine.FontDC} {
		p := source()
		p.Layout.FontID = id
		plan, err := Build(font(t), p, e, Placement{})
		if err != nil {
			t.Fatal(id, err)
		}
		if len(plan.Groups) != 2 || len(plan.Objects) != 4 {
			t.Fatal("caption detached from its line", plan.Groups)
		}
		if strings.Contains(plan.Objects[0].Cells[0].Text, "\n") {
			t.Fatal("single-line caption wrapped")
		}
		blanks := 0
		for _, preview := range plan.Preview {
			for _, c := range preview.Cells {
				if c.Text != "" && c.FontSize+1e-7 < p.Layout.FontSize*.7 {
					t.Fatal("source table dropped below readability floor", id, c.FontSize)
				}
				if strings.Contains(c.Text, "        mm") {
					blanks++
				}
			}
		}
		if blanks != 3 {
			t.Fatal("source answer blanks collapsed", blanks)
		}
		assertImportRendersAndExports(t, p, plan)
	}
}
