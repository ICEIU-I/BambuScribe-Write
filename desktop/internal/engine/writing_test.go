package engine

import (
	"strings"
	"testing"
)

func testWriting() WritingSettings {
	up, down := 7.0, 3.4
	return WritingSettings{PenUp: &up, PenDown: &down, OriginMode: "manual", Sound: true}
}

func TestWritingSettingsAndAutomaticOrigin(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	d, _ := f.Layout("你好", s)
	if _, err := ExportPage(d, 0, WritingSettings{OriginMode: "auto"}); err == nil {
		t.Fatal("unset Z must block execution")
	}
	w := testWriting()
	w.OriginMode = "auto"
	w.OffsetX = 2
	w.OffsetY = -3
	gcode, err := GenerateGCode(d.Pages[0], s, w)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(gcode, "G91") {
		t.Fatal("automatic origin should use absolute XY")
	}
	if !strings.Contains(gcode, "G1 Z7.000") || !strings.Contains(gcode, "G1 Z3.400") {
		t.Fatal("configured Z heights missing")
	}
	w.OffsetX = 256
	if _, err = GenerateGCode(d.Pages[0], s, w); err == nil {
		t.Fatal("automatic origin must reject out-of-bounds placement")
	}
	w = testWriting()
	*w.PenUp = *w.PenDown
	if err = w.Validate(); err == nil {
		t.Fatal("pen must clear the writing plane")
	}
}

func TestRequestedPageDefaultsAndStableRandomRows(t *testing.T) {
	s := DefaultLayout()
	if s.TemplateName != "新建笔记" || s.PaperWidth != 163 || s.PaperHeight != 240 || s.MarginTop+s.MarginBottom+s.MarginLeft+s.MarginRight != 0 || s.LineCount != 16 || s.LineMode != "fixed" {
		t.Fatal("page defaults differ from requested values")
	}
	if s.FontSize != 16.5 || s.FontSize/s.LineHeight != 1.1 {
		t.Fatal("default font must use the requested 110% row height")
	}
	s.LineMode = "random"
	f := testFont(t)
	text := strings.Repeat("你好，世界。\n", 120)
	a, err := f.Layout(text, s)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := f.Layout(text, s)
	if a.ID != b.ID || len(a.Pages) != len(b.Pages) {
		t.Fatal("random pagination must stay stable")
	}
	for i, p := range a.Pages {
		if p.LineCount < 15 || p.LineCount > 17 || p.LineCount != b.Pages[i].LineCount {
			t.Fatal("random rows out of range or unstable")
		}
	}
}
