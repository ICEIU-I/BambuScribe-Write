package engine

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestOnlyTableBordersUseSeventyPercentFeedAfterErasureAndReload(t *testing.T) {
	f := testFont(t)
	for _, style := range []string{"grid", "three-line", "horizontal", "none"} {
		t.Run(style, func(t *testing.T) {
			s := DefaultLayout()
			s.FontSize = 6
			table := sampleTable()
			table.Y, table.Style = 60, style
			caption := sampleTextBox()
			caption.Page, caption.Y, caption.Cells[0].Text = 0, 120, "图一 ____"
			im := textBoxImage(t)
			im.Y, im.Width, im.Height = 180, 40, 10
			e := PageErasure{ID: "split-grid", X: table.X + 10, Y: table.Y - 1, Width: 5, Height: 2}
			doc, err := f.LayoutWithErasures("正文Ab", s, []ImageAsset{im}, []TableAsset{caption, table}, []PageErasure{e})
			if err != nil {
				t.Fatal(err)
			}
			if style != "none" && (len(doc.Erasures) != 1 || len(*doc.Erasures[0].Cuts) != 1) {
				t.Fatal("fixture did not split exactly one grid stroke")
			}
			// Render metadata must survive serialized previews as well as saved cuts.
			raw, _ := json.Marshal(doc)
			var restored Document
			if err := json.Unmarshal(raw, &restored); err != nil {
				t.Fatal(err)
			}
			assertWritingReplay(t, &restored)
			for _, mode := range []string{"auto", "manual"} {
				w := testWriting()
				w.OriginMode = mode
				data, err := ExportPage(&restored, 0, w)
				if err != nil {
					t.Fatal(err)
				}
				grid := tableLines(table, table.RowHeights)
				var previous Point
				penDown, slow, fast := false, 0, 0
				for _, line := range strings.Split(string(data), "\n") {
					if !strings.HasPrefix(line, "G1 ") {
						continue
					}
					values := map[byte]float64{}
					for _, field := range strings.Fields(line)[1:] {
						values[field[0]], err = strconv.ParseFloat(field[1:], 64)
						if err != nil {
							t.Fatal(err)
						}
					}
					want := 16000.0
					if z, ok := values['Z']; ok {
						penDown = z == *w.PenDown
					} else if _, ok := values['X']; ok {
						point := Point{values['X'], values['Y']}
						if penDown {
							for _, stroke := range grid {
								a, b := stroke[0], stroke[len(stroke)-1]
								a.X, a.Y, b.X, b.Y = a.X+table.X, a.Y+table.Y, b.X+table.X, b.Y+table.Y
								ax, ay, _ := machineTarget(a, s, w)
								bx, by, _ := machineTarget(b, s, w)
								on := func(p Point) bool {
									left, right := float64(min(ax, bx))/1000, float64(max(ax, bx))/1000
									top, bottom := float64(min(ay, by))/1000, float64(max(ay, by))/1000
									return p.X >= left-.00051 && p.X <= right+.00051 && p.Y >= top-.00051 && p.Y <= bottom+.00051
								}
								if on(previous) && on(point) {
									want = 11200
								}
							}
							if want == 11200 {
								slow++
							} else {
								fast++
							}
						}
						previous = point
					}
					if math.Abs(values['F']-want) > 1e-8 {
						t.Fatalf("%s: feed in %q = %.0f, want %.0f", mode, line, values['F'], want)
					}
				}
				if fast == 0 || style != "none" && slow != len(grid)+1 || style == "none" && slow != 0 {
					t.Fatalf("missing ordinary strokes or cut grid fragments: slow=%d fast=%d", slow, fast)
				}
			}
		})
	}
}
