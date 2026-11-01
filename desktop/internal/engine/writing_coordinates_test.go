package engine

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// Interpret the emitted coordinate modes to verify physical displacement,
// rather than reusing the exporter's coordinate conversion in the assertions.
func xyMoves(t *testing.T, gcode string, start Point) []Point {
	t.Helper()
	position, offset := start, Point{}
	var moves []Point
	for _, line := range strings.Split(gcode, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "G91" {
			t.Fatal("Chixie's A1 format uses absolute XY")
		}
		if fields[0] != "G92" && fields[0] != "G0" && fields[0] != "G1" {
			continue
		}
		xy := false
		for _, f := range fields[1:] {
			if f[0] != 'X' && f[0] != 'Y' {
				if fields[0] == "G92" && f[0] == 'Z' {
					t.Fatal("XY origin must not overwrite absolute Z calibration")
				}
				continue
			}
			n, err := strconv.ParseFloat(f[1:], 64)
			if err != nil {
				t.Fatal(err)
			}
			xy = true
			axis, shift := &position.X, &offset.X
			if f[0] == 'Y' {
				axis, shift = &position.Y, &offset.Y
			}
			if fields[0] == "G92" {
				*shift = *axis - n
			} else {
				*axis = n + *shift
			}
		}
		if xy && fields[0] != "G92" {
			moves = append(moves, position)
		}
	}
	return moves
}

func TestChixieFrameOriginModesOffsetsAndMargins(t *testing.T) {
	settings := DefaultLayout()
	settings.MarginLeft, settings.MarginTop = 5, 10
	page := Page{Number: 1, Strokes: []Stroke{{{X: 20, Y: 30}, {X: 70, Y: 80}}}}
	for _, mode := range []string{"auto", "manual"} {
		t.Run(mode, func(t *testing.T) {
			writing := testWriting()
			writing.OffsetX, writing.OffsetY, writing.OriginMode = 2.5, -3.5, mode
			gcode, err := GenerateGCode(page, settings, writing)
			if err != nil {
				t.Fatal(err)
			}
			start := Point{X: 32, Y: 250}
			want := []Point{{X: 17.5, Y: 223.5}, {X: 67.5, Y: 173.5}, {}}
			if mode == "auto" {
				// The frame origin is labelled X0/Y240; finish back at startup.
				want = []Point{{X: 49.5, Y: 233.5}, {X: 99.5, Y: 183.5}, start}
				if strings.Count(gcode, "G92 X0 Y240.000\n") != 1 {
					t.Fatal("missing or repeated XY origin reset")
				}
			} else if strings.Contains(gcode, "G92") {
				t.Fatal("None must retain the printer's current XY coordinate system")
			}
			got := xyMoves(t, gcode, start)
			if len(got) != len(want) {
				t.Fatalf("got %v want %v", got, want)
			}
			for i := range want {
				if math.Abs(got[i].X-want[i].X) > .0005 || math.Abs(got[i].Y-want[i].Y) > .0005 {
					t.Fatalf("move %d: got %v want %v", i, got[i], want[i])
				}
			}
			if writing.OffsetX != 2.5 || writing.OffsetY != -3.5 || *writing.PenUp != 7 || *writing.PenDown != 3.4 || writing.OriginMode != mode || !writing.Sound {
				t.Fatal("export modified configured writing values")
			}
		})
	}
}

func TestChixieTravelValidationIncludesFinalParkingAndSpans(t *testing.T) {
	page := Page{Number: 1, Strokes: []Stroke{{{X: 20, Y: 30}, {X: 40, Y: 50}}}}
	settings, writing := DefaultLayout(), testWriting()
	writing.OffsetX = -50
	if _, err := GenerateGCode(page, settings, writing); err == nil {
		t.Fatal("None allowed negative absolute machine coordinates")
	}
	writing.OriginMode = "auto"
	if _, err := GenerateGCode(page, settings, writing); err != nil {
		t.Fatal("auto origin rejected a valid negative displacement", err)
	}
	writing.OffsetX = 0
	settings.PaperHeight = 300
	gcode, err := GenerateGCode(page, settings, writing)
	if err != nil {
		t.Fatal("paper height alone must not create an unreachable parking move", err)
	}
	start := Point{X: 0, Y: 256}
	moves := xyMoves(t, gcode, start)
	if moves[len(moves)-1] != start {
		t.Fatal("automatic parking did not return to the starting position")
	}
	settings = DefaultLayout()
	writing.OffsetX = -200
	page.Strokes = []Stroke{{{X: 0, Y: 30}, {X: 400, Y: 50}}}
	if _, err := GenerateGCode(page, settings, writing); err == nil {
		t.Fatal("each target is reachable but the 400 mm span is not")
	}
	page.Strokes = []Stroke{{{X: 0, Y: 257}, {X: 1, Y: 258}}}
	writing.OffsetX = 0
	if _, err := GenerateGCode(page, settings, writing); err == nil {
		t.Fatal("small ink span cannot make a 258 mm trip from the origin valid")
	}
}

func TestA4FrameFitsA1WithoutChangingLayoutOrCalibration(t *testing.T) {
	s := DefaultLayout()
	s.PaperWidth, s.PaperHeight = 210, 297
	s.MarginLeft, s.MarginRight, s.MarginTop, s.MarginBottom = 25.4, 25.4, 25.4, 25.4
	page := Page{Number: 1, Strokes: []Stroke{
		{{X: 25.4, Y: 25.4}, {X: 184.6, Y: 271.6}},
		// An image outside the left margin must keep its relative position.
		{{X: 20.4, Y: 30.4}, {X: 26.4, Y: 40.4}},
	}}
	w := DefaultWriting()
	gcode, err := GenerateGCode(page, s, w)
	if err != nil {
		t.Fatal("159.2 x 246.2 mm frame should fit A1", err)
	}
	start := Point{X: 40, Y: 252}
	want := []Point{start, {X: 199.2, Y: 5.8}, {X: 35, Y: 247}, {X: 41, Y: 237}, start}
	got := xyMoves(t, gcode, start)
	if len(got) != len(want) {
		t.Fatalf("moves=%v want=%v", got, want)
	}
	for i, p := range got {
		if math.Abs(p.X-want[i].X) > .0005 || math.Abs(p.Y-want[i].Y) > .0005 {
			t.Fatalf("move %d: got %v want %v", i, p, want[i])
		}
		if p.X < 0 || p.Y < 0 || p.X > BedSize || p.Y > BedSize {
			t.Fatalf("physical move outside A1: %v", p)
		}
	}
	if w.OffsetX != 0 || w.OffsetY != 0 || *w.PenUp != 7 || *w.PenDown != 3.4 || w.OriginMode != "auto" || !w.Sound || s.MarginTop != 25.4 || s.PaperHeight != 297 || page.Strokes[0][1].Y != 271.6 {
		t.Fatal("export changed calibration or page geometry")
	}
	// 256 mm of displacement is valid; a genuine excess still blocks export.
	page.Strokes[0][1].Y = 281.4
	if _, err := GenerateGCode(page, s, w); err != nil {
		t.Fatal("rejected exact 256 mm travel", err)
	}
	page.Strokes[0][1].Y = 281.401
	if _, err := GenerateGCode(page, s, w); err == nil {
		t.Fatal("allowed travel beyond 256 mm")
	}
	w.OriginMode = "manual"
	if _, err := GenerateGCode(page, s, w); err == nil {
		t.Fatal("None must still enforce absolute machine bounds")
	}
}

func TestCompletionSoundFollowsThePreservedSetting(t *testing.T) {
	page := Page{Number: 1, Strokes: []Stroke{{{X: 20, Y: 30}, {X: 40, Y: 50}}}}
	writing := testWriting()
	for _, enabled := range []bool{true, false} {
		writing.Sound = enabled
		gcode, err := GenerateGCode(page, DefaultLayout(), writing)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(gcode, "M1006") != enabled {
			t.Fatal("completion sound did not follow the setting")
		}
		park := strings.LastIndex(gcode, "G1 X0.000 Y0.000")
		if park < 0 {
			t.Fatal("missing final lifted-pen parking move")
		}
		if enabled && strings.Index(gcode, "M1006 S1") < park {
			t.Fatal("sound must wait until writing is complete")
		}
		if !strings.HasSuffix(gcode, "M400\nM18 X Y Z\n") {
			t.Fatal("motors were not released after finishing")
		}
	}
}
