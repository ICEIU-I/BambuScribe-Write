package engine

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestExportedMotionUsesRaisedFeedWithoutSlowPenCycles(t *testing.T) {
	for _, mode := range []string{"auto", "manual"} {
		t.Run(mode, func(t *testing.T) {
			settings, writing := DefaultLayout(), testWriting()
			writing.OriginMode = mode
			doc := &Document{Settings: settings, Pages: []Page{{Number: 1, Strokes: []Stroke{
				{{X: 20, Y: 30}, {X: 25, Y: 30}, {X: 25, Y: 35}},
				{{X: 30, Y: 35}, {X: 35, Y: 40}},
			}}}}
			data, err := ExportPage(doc, 0, writing)
			if err != nil {
				t.Fatal(err)
			}
			feed, z, cycleSeconds := 0.0, 7.0, 0.0
			counts := map[string]int{}
			for _, line := range strings.Split(string(data), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 0 {
					continue
				}
				if fields[0] == "G4" {
					t.Fatal("unexpected pause between strokes")
				}
				if fields[0] != "G0" && fields[0] != "G1" {
					continue
				}
				if fields[0] == "G0" {
					t.Fatalf("Chixie uses G1 for raised-pen travel and writing: %q", line)
				}
				nextZ, hasZ := z, false
				for _, field := range fields[1:] {
					if field[0] != 'F' && field[0] != 'Z' {
						continue
					}
					value, err := strconv.ParseFloat(field[1:], 64)
					if err != nil {
						t.Fatalf("invalid motion parameter in %q: %v", line, err)
					}
					if field[0] == 'F' {
						feed = value
					} else {
						nextZ, hasZ = value, true
					}
				}
				if feed != 16000 {
					t.Fatalf("unexpected modal feed %.0f mm/min in %q; expected 16000", feed, line)
				}
				if hasZ {
					if nextZ == 50 {
						if counts["pen"] != 5 || z != 7 {
							t.Fatal("high lift must follow all writing pen cycles")
						}
						counts["return"]++
						z = nextZ
						continue
					}
					cycleSeconds += math.Abs(nextZ-z) / math.Min(feed/60, 30)
					if nextZ != 7 && nextZ != 3.4 {
						t.Fatalf("changed calibrated pen position to %.3f", nextZ)
					}
					counts["pen"]++
					z = nextZ
				} else if z == 3.4 {
					counts["writing"]++
				} else {
					counts["travel"]++
				}
			}
			if counts["pen"] != 5 || counts["writing"] != 3 || counts["travel"] != 3 || counts["return"] != 1 || z != 50 {
				t.Fatalf("lost stroke boundaries or raised-pen travel: %v, final Z %.3f", counts, z)
			}
			// Two complete 7/3.4 mm cycles cover 14.4 mm. Even with F16000,
			// A1 caps Z at 30 mm/s: 0.48 s before acceleration/deceleration.
			if math.Abs(cycleSeconds-0.48) > 1e-9 {
				t.Fatalf("pen cycles still use slow feed: %.3f seconds", cycleSeconds)
			}
		})
	}
}
