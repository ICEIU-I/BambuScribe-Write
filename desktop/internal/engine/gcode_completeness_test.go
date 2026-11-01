package engine

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// Replay the exported XY origin and pen state to check what would actually be
// drawn. A preview-only assertion would miss a cropped or pen-up export.
func replayWrittenStrokes(t *testing.T, code string, start Point, penDown float64) []Stroke {
	t.Helper()
	position, shift := start, Point{}
	z := math.NaN()
	var strokes []Stroke
	for _, line := range strings.Split(code, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || (fields[0] != "G92" && fields[0] != "G1") {
			continue
		}
		xy, hasZ, nextZ := false, false, z
		for _, field := range fields[1:] {
			value, err := strconv.ParseFloat(field[1:], 64)
			if err != nil {
				t.Fatal(err)
			}
			switch field[0] {
			case 'X', 'Y':
				axis, offset := &position.X, &shift.X
				if field[0] == 'Y' {
					axis, offset = &position.Y, &shift.Y
				}
				if fields[0] == "G92" {
					*offset = *axis - value
				} else {
					*axis = value + *offset
					xy = true
				}
			case 'Z':
				if fields[0] == "G92" {
					t.Fatal("XY origin reset changed Z calibration")
				}
				hasZ, nextZ = true, value
			}
		}
		if xy && z == penDown {
			strokes[len(strokes)-1] = append(strokes[len(strokes)-1], position)
		}
		if hasZ {
			if nextZ == penDown && z != penDown {
				strokes = append(strokes, Stroke{position})
			}
			z = nextZ
		}
	}
	return strokes
}

func TestA4LowerRowsRetainEveryWrittenPoint(t *testing.T) {
	f := testFont(t)
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight = 210, 297
	s.MarginLeft, s.MarginRight, s.MarginTop, s.MarginBottom = 25.4, 25.4, 25.4, 25.4
	s.LineCount, s.FontSize, s.LineHeight = 30, 8.206666666666667, 8.206666666666667
	s.LetterSpacing, s.LineSpacing = 85, 85
	text := strings.Repeat("光电效应\n", 32) +
		"射光频率；m为电子的质量；v为光电子逸出金属表面的\n" +
		"初速度；Ws为受光照射的金属材料表面的逸出功.\n" +
		"式(1)就是著名的爱因斯坦光电效应方程.在式(1)中,mv"
	doc, err := f.LayoutWithImages(text, s, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 || doc.Pages[0].Text != text || len(doc.Pages[0].Images) != 0 {
		t.Fatal("lower rows were removed, repaginated or covered")
	}
	w := testWriting()
	w.OriginMode = "auto"
	data, err := ExportPage(doc, 0, w)
	if err != nil {
		t.Fatal(err)
	}
	start := Point{X: 30, Y: 250}
	got := replayWrittenStrokes(t, string(data), start, *w.PenDown)
	want := doc.Pages[0].TextStrokes
	if len(got) != len(want) {
		t.Fatalf("written strokes=%d, preview strokes=%d", len(got), len(want))
	}
	lowerPoints := 0
	for i, stroke := range want {
		if len(got[i]) != len(stroke) {
			t.Fatalf("stroke %d lost points: written=%d, preview=%d", i, len(got[i]), len(stroke))
		}
		for j, p := range stroke {
			// The operator registers the pen at the frame's upper-left corner.
			expected := Point{X: start.X + p.X - s.MarginLeft, Y: start.Y - (p.Y - s.MarginTop)}
			if math.Abs(got[i][j].X-expected.X) > .00051 || math.Abs(got[i][j].Y-expected.Y) > .00051 {
				t.Fatalf("stroke %d point %d: written=%v, expected=%v", i, j, got[i][j], expected)
			}
			if expected.X < 0 || expected.X > BedSize || expected.Y < 0 || expected.Y > BedSize {
				t.Fatalf("physical target outside A1: %v", expected)
			}
			if p.Y-s.MarginTop > 227 {
				lowerPoints++
			}
		}
	}
	if lowerPoints < 100 {
		t.Fatal("fixture did not exercise the cut-off row's lower half and following rows")
	}
}
