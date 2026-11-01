package engine

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestDiagramStraightRunPreservesJunctionsAndRealBends(t *testing.T) {
	line := Stroke{}
	for i := 0; i <= 160; i++ {
		y := 25. + .27*float64(i) + .9*math.Sin(float64(i)*.3)
		if i == 0 || i == 160 {
			y = 25. + .27*float64(i)
		}
		line = append(line, Point{float64(i), y})
	}
	got := fitDiagramStroke(line)
	if len(got) != 2 || got[0] != line[0] || got[1] != line[len(line)-1] {
		t.Fatal("straight wire retained raster wobble or moved a junction", len(got))
	}
	corner := resampleImageStroke(Stroke{{0, 0}, {80, 0}, {80, 90}}, 1)
	got = fitDiagramStroke(corner)
	if len(got) != 3 || got[1] != (Point{80, 0}) {
		t.Fatal("corner was rounded or cut", got)
	}
	curve := Stroke{}
	for i := 0; i <= 180; i++ {
		a := float64(i) * math.Pi / 180
		curve = append(curve, Point{50 * math.Cos(a), 50 * math.Sin(a)})
	}
	got = fitDiagramStroke(curve)
	if len(got) < 8 {
		t.Fatal("real curve flattened")
	}
	for _, p := range curve {
		if distanceToStroke(p, got) > 1.6 {
			t.Fatal("curve moved outside raster tolerance")
		}
	}
}

func TestCircleArcsStayAttachedToWiresAndDoNotCloseGaps(t *testing.T) {
	loop := Stroke{}
	for i := 0; i <= 64; i++ {
		a := float64(i) * 2 * math.Pi / 64
		r := 8. + .3*math.Sin(3*a)
		loop = append(loop, Point{50 + r*math.Cos(a), 40 + r*math.Sin(a)})
	}
	loop[len(loop)-1] = loop[0]
	input := []Stroke{loop[:17], loop[16:33], loop[32:], {{20, 40}, loop[32]}, {loop[0], {80, 40}}, {{90, 40}, {110, 40}}}
	paths, circles := regularizeImageCircles(input)
	if len(circles) != 1 || len(paths) != 3 {
		t.Fatalf("did not recover split ring: %d circles, %d other paths", len(circles), len(paths))
	}
	if distanceToStroke(paths[0][1], circles[0]) > .08 || distanceToStroke(paths[1][0], circles[0]) > .08 {
		t.Fatal("circle detached from its wires")
	}
	if paths[1][1] != (Point{80, 40}) || paths[2][0] != (Point{90, 40}) {
		t.Fatal("open switch gap closed")
	}
	center, radius, ok := fitImageCircle(circles[0])
	if !ok || distance(center, Point{50, 40}) > .2 || math.Abs(radius-8) > .2 {
		t.Fatal("bad circle reconstruction", center, radius)
	}
}

func TestCircleFitKeepsBoxesOvalsAndOpenArcs(t *testing.T) {
	for _, size := range []float64{5, 10, 20} {
		square := resampleImageStroke(Stroke{{-size, -size}, {size, -size}, {size, size}, {-size, size}, {-size, -size}}, 1)
		for _, angle := range []float64{0, .4} {
			rotated := Stroke{}
			for _, p := range square {
				rotated = append(rotated, Point{p.X*math.Cos(angle) - p.Y*math.Sin(angle), p.X*math.Sin(angle) + p.Y*math.Cos(angle)})
			}
			if _, _, ok := fitImageCircle(rotated); ok {
				t.Fatal("square rounded into circle", size, angle)
			}
		}
	}
	oval := Stroke{}
	for i := 0; i <= 80; i++ {
		a := float64(i) * 2 * math.Pi / 80
		oval = append(oval, Point{16 * math.Cos(a), 8 * math.Sin(a)})
	}
	oval[len(oval)-1] = oval[0]
	if _, _, ok := fitImageCircle(oval); ok {
		t.Fatal("ellipse rounded")
	}
	if _, _, ok := fitImageCircle(oval[:70]); ok {
		t.Fatal("open arc closed")
	}
}

func TestHallSwitchChineseLabelsAndWireGeometry(t *testing.T) {
	raw, err := os.ReadFile("testdata/hall-switch.png")
	if err != nil {
		t.Fatal(err)
	}
	var boxes []OCRBox
	data, err := os.ReadFile("testdata/hall-switch-ocr.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &boxes); err != nil {
		t.Fatal(err)
	}
	asset := ImageAsset{ID: "switch", X: 5, Y: 8, Width: 150, Height: 150. * 526 / 1354, Data: "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)}
	f := testFont(t)
	asset.TextRegions = resolveImageOCR(f, nil, boxes, nil, nil, 1354, 526, asset)
	render, err := imageStrokes(f, asset, DefaultLayout())
	if err != nil {
		t.Fatal(err)
	}
	han := 0
	for _, r := range render.TextRegions {
		for _, c := range r.Text {
			if c >= '\u3400' && c <= '\u9fff' {
				han++
				if r.Mode != "handwriting" || len(r.Strokes) == 0 {
					t.Fatal("read Chinese label was not rendered", r.Text)
				}
				break
			}
		}
	}
	if han != 10 {
		t.Fatal("fixture lost a Chinese label", han)
	}
	// Long wires remain straight through their central portions. Source rings
	// and crossings remain independent; writing must not bridge switch gaps.
	pixel := func(p Point) Point {
		return Point{(p.X - asset.X) / asset.Width * 1353, (p.Y - asset.Y) / asset.Height * 525}
	}
	for _, target := range []Point{{955, 266}, {805, 250}, {1070, 323}, {807, 325}} {
		found := false
		for _, s := range render.GraphicStrokes {
			for i := 1; i < len(s); i++ {
				a, b := pixel(s[i-1]), pixel(s[i])
				if distance(a, b) >= 80 && pointLineDistance(target, a, b) < 3 {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("real wire still fragmented around", target)
		}
	}
	doc, err := f.LayoutWithImages("", DefaultLayout(), []ImageAsset{asset})
	if err != nil {
		t.Fatal(err)
	}
	want, err := GenerateGCode(doc.Pages[0], doc.Settings, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	exported, err := ExportPage(doc, 0, testWriting())
	if err != nil || string(exported) != want {
		t.Fatal("export differs from visible geometry", err)
	}
}
