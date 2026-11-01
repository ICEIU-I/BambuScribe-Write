package engine

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestPageHandoffKeepsPositionForManuallyStartedPages(t *testing.T) {
	for _, mode := range []string{"auto", "manual"} {
		for _, sound := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/sound=%t", mode, sound), func(t *testing.T) {
				s := DefaultLayout()
				s.PaperHeight = 297
				s.MarginLeft, s.MarginTop = 25.4, 25.4
				w := testWriting()
				w.OriginMode, w.Sound = mode, sound
				w.OffsetX, w.OffsetY = 2.5, -3.5
				start := Point{X: 32, Y: 250}
				park := Point{}
				if mode == "auto" {
					park = start
				}
				var combined strings.Builder
				for i := range 3 {
					// Each page has different ink and a different final position.
					page := Page{Number: i + 1, Strokes: []Stroke{{
						{X: 40 + float64(i)*10, Y: 75},
						{X: 90, Y: 220 + float64(i)*5},
					}}}
					data, err := ExportPage(&Document{Settings: s, Pages: []Page{page}}, 0, w)
					if err != nil {
						t.Fatal(err)
					}
					code := string(data)
					z, lastXYHeight := math.NaN(), math.NaN()
					settled, lastXYSettled := false, false
					enabled := false
					for _, line := range strings.Split(code, "\n") {
						fields := strings.Fields(strings.SplitN(line, ";", 2)[0])
						if len(fields) == 0 {
							continue
						}
						switch fields[0] {
						case "M17":
							enabled = true
						case "M400":
							settled = true
						case "M18", "M84", "G28", "M0", "M1", "M25":
							t.Fatalf("page must finish without releasing, homing or pausing: %s", line)
						case "G1":
							if !enabled {
								t.Fatal("movement before enabling motors")
							}
							for _, field := range fields[1:] {
								if field[0] == 'Z' {
									settled = false
									z, err = strconv.ParseFloat(field[1:], 64)
									if err != nil {
										t.Fatal(err)
									}
								}
								if field[0] == 'X' || field[0] == 'Y' {
									lastXYHeight = z
									lastXYSettled = settled
								}
							}
						}
						if line == "M400 U1" {
							t.Fatal("page must complete, not stay paused in a running job")
						}
					}
					if !enabled || z != 50 || lastXYHeight != 50 || !lastXYSettled || !strings.HasSuffix(code, "M400\n") {
						t.Fatal("page must finish lifting to Z50 before parking, then hold height and motor position")
					}
					combined.WriteString(code)
					moves := xyMoves(t, combined.String(), start)
					if moves[len(moves)-1] != park {
						t.Fatalf("page %d parked at %v, want %v", page.Number, moves[len(moves)-1], park)
					}
					written := replayWrittenStrokes(t, combined.String(), start, *w.PenDown)
					if len(written) != i+1 {
						t.Fatal("parking drew a line or a later page lost its stroke")
					}
					want := Point{X: page.Strokes[0][0].X - s.MarginLeft + w.OffsetX, Y: s.PaperHeight - (page.Strokes[0][0].Y - s.MarginTop + w.OffsetY)}
					if mode == "auto" {
						want.X += start.X
						want.Y += start.Y - s.PaperHeight
					}
					if got := written[i][0]; math.Abs(got.X-want.X) > .00051 || math.Abs(got.Y-want.Y) > .00051 {
						t.Fatalf("page %d drifted: first point %v, want %v", page.Number, got, want)
					}
				}
			})
		}
	}
}

func TestPageReturnDoesNotLowerAHigherPenUpHeight(t *testing.T) {
	w := testWriting()
	*w.PenUp = 80
	code, err := GenerateGCode(Page{Number: 1, Strokes: []Stroke{{{X: 20, Y: 30}, {X: 40, Y: 50}}}}, DefaultLayout(), w)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(code, "G1 Z80.000 F16000\nM400\nG1 X0.000 Y0.000 F16000\n") || strings.Contains(code, "G1 Z50.000") {
		t.Fatal("page return must preserve a configured pen-up height above Z50")
	}
}
