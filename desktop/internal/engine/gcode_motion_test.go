package engine

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestWritingInitialisesMotionBeforeMovingAndKeepsCalibratedZ(t *testing.T) {
	page := Page{Number: 1, Strokes: []Stroke{
		{{X: 20, Y: 30}, {X: 70, Y: 80}},
		{{X: 70, Y: 220}, {X: 20, Y: 225}},
	}}
	for _, mode := range []string{"auto", "manual"} {
		t.Run(mode, func(t *testing.T) {
			w := testWriting()
			w.OriginMode = mode
			code, err := GenerateGCode(page, DefaultLayout(), w)
			if err != nil {
				t.Fatal(err)
			}
			// Deliberately start with slow inherited limits and multipliers.
			feedScale, accelScale, accel := .5, .3, 100.0
			speedCaps, accelCaps := map[byte]float64{'X': 10, 'Y': 10, 'Z': 1}, map[byte]float64{'X': 50, 'Y': 50, 'Z': 20}
			seen := map[string]bool{}
			zMoves, xyMoves := 0, 0
			for _, line := range strings.Split(code, "\n") {
				fields := strings.Fields(line)
				if len(fields) == 0 {
					continue
				}
				command := fields[0]
				if command != "M220" && command != "M201.2" && command != "M201" && command != "M203" && command != "M204" && command != "G1" {
					continue
				}
				values := map[byte]float64{}
				for _, field := range fields[1:] {
					v, err := strconv.ParseFloat(field[1:], 64)
					if err != nil {
						t.Fatal(err)
					}
					values[field[0]] = v
				}
				switch command {
				case "M220":
					feedScale = values['S'] / 100
				case "M201.2":
					accelScale = values['K']
				case "M201":
					for axis, v := range values {
						accelCaps[axis] = v
					}
				case "M203":
					for axis, v := range values {
						speedCaps[axis] = v
					}
				case "M204":
					accel = values['S']
				case "G1":
					if len(seen) != 5 {
						t.Fatal("movement occurred before speed state was initialised")
					}
					if feedScale != 1 || accelScale != 1 || accel != 12000 {
						t.Fatal("inherited speed state survived")
					}
					if values['F'] != 16000 {
						t.Fatalf("missing raised feed rate: %s", line)
					}
					if z, ok := values['Z']; ok {
						zMoves++
						if z != 3.4 && z != 7 && z != 50 {
							t.Fatalf("calibrated Z changed: %s", line)
						}
						if z == 50 && (zMoves != 6 || xyMoves != 4) {
							t.Fatal("high lift must occur only before the final return")
						}
						if math.Min(values['F']/60*feedScale, speedCaps['Z']) != 30 || math.Min(accel*accelScale, accelCaps['Z']) != 1500 {
							t.Fatal("incorrect A1 Z motion limits")
						}
					} else {
						xyMoves++
						for _, axis := range []byte{'X', 'Y'} {
							if speedCaps[axis] != 500 || accelCaps[axis] != 12000 {
								t.Fatal("incorrect XY limits")
							}
						}
					}
				}
				if command != "G1" {
					seen[command] = true
				}
			}
			if xyMoves != 5 || zMoves != 6 {
				t.Fatal("stroke moves or pen lifts changed")
			}
			for _, forbidden := range []string{"M211", "G29", "G92 Z", "M500", "M201 E", "M203 E"} {
				if strings.Contains(code, forbidden) {
					t.Fatalf("unexpected unrelated command: %s", forbidden)
				}
			}
		})
	}
}
