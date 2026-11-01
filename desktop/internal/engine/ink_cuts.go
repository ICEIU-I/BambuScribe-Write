package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

// InkLayer identifies original vectors independently of their paper position.
// Body Part is a UTF-16 character offset; object parts are graphics or cells.
type InkLayer struct {
	ObjectID string   `json:"objectId"`
	Part     string   `json:"part"`
	Source   string   `json:"source"`
	Strokes  []Stroke `json:"strokes"`
}
type InkCut struct {
	ObjectID string `json:"objectId"`
	Part     string `json:"part"`
	Source   string `json:"source"`
	Stroke   int    `json:"stroke"`
	// Ranges use vertex index + segment fraction, never paper coordinates.
	Ranges [][2]float64 `json:"ranges"`
}

func inkSource(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}
func cutLayer(layer InkLayer, erasures []PageErasure) []Stroke {
	cuts := map[int][][2]float64{}
	for _, e := range erasures {
		if e.Cuts == nil {
			continue
		}
		for _, c := range *e.Cuts {
			if c.ObjectID == layer.ObjectID && c.Part == layer.Part && c.Source == layer.Source {
				cuts[c.Stroke] = append(cuts[c.Stroke], c.Ranges...)
			}
		}
	}
	out := []Stroke{}
	for i, stroke := range layer.Strokes {
		ranges := cuts[i]
		if len(ranges) == 0 {
			out = append(out, stroke)
			continue
		}
		sort.Slice(ranges, func(i, j int) bool { return ranges[i][0] < ranges[j][0] })
		end := float64(len(stroke) - 1)
		at := func(v float64) Point {
			n := min(int(v), len(stroke)-2)
			t := v - float64(n)
			return Point{stroke[n].X + t*(stroke[n+1].X-stroke[n].X), stroke[n].Y + t*(stroke[n+1].Y-stroke[n].Y)}
		}
		add := func(lo, hi float64) {
			if hi-lo < 1e-10 || len(stroke) < 2 {
				return
			}
			path := Stroke{at(lo)}
			for j := int(math.Floor(lo)) + 1; float64(j) < hi; j++ {
				path = append(path, stroke[j])
			}
			out = append(out, append(path, at(hi)))
		}
		cursor := 0.0
		for _, c := range ranges {
			lo, hi := min(end, c[0]), min(end, c[1])
			if lo > cursor {
				add(cursor, lo)
			}
			cursor = max(cursor, hi)
		}
		add(cursor, end)
	}
	return out
}
func captureInk(e PageErasure, layers []InkLayer) []InkCut {
	out := []InkCut{}
	for _, layer := range layers {
		if layer.ObjectID != "" && e.ObjectIDs != nil && !containsID(*e.ObjectIDs, layer.ObjectID) {
			continue
		}
		for si, stroke := range layer.Strokes {
			ranges := [][2]float64{}
			for i := 1; i < len(stroke); i++ {
				for _, c := range paperIntervals(stroke[i-1], stroke[i], e) {
					ranges = append(ranges, [2]float64{float64(i-1) + c.from, float64(i-1) + c.to})
				}
			}
			if len(ranges) > 0 {
				out = append(out, InkCut{layer.ObjectID, layer.Part, layer.Source, si, ranges})
			}
		}
	}
	return out
}
func containsID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
func paperIntervals(a, b Point, e PageErasure) []eraseInterval {
	out := []eraseInterval{}
	add := func(v eraseInterval, ok bool) {
		if ok {
			out = append(out, v)
		}
	}
	if len(e.Path) == 0 {
		add(rectangleInterval(a, b, e.X, e.Y, e.X+e.Width, e.Y+e.Height))
		return out
	}
	for i, p := range e.Path {
		add(circleInterval(a, b, p, e.Radius))
		if i == 0 {
			continue
		}
		q := e.Path[i-1]
		dx, dy := p.X-q.X, p.Y-q.Y
		length := math.Hypot(dx, dy)
		if length < 1e-10 {
			continue
		}
		transform := func(t Point) Point {
			return Point{((t.X-q.X)*dx + (t.Y-q.Y)*dy) / length, (-(t.X-q.X)*dy + (t.Y-q.Y)*dx) / length}
		}
		add(rectangleInterval(transform(a), transform(b), 0, -e.Radius, length, e.Radius))
	}
	return out
}
func validateInkCuts(cuts *[]InkCut, total *int) error {
	if cuts == nil {
		return nil
	}
	for _, c := range *cuts {
		if len(c.ObjectID) > 1000 || len(c.Part) > 1000 || c.Part == "" || len(c.Source) > 1000 || c.Source == "" || c.Stroke < 0 || c.Stroke > 2000000 {
			return fmt.Errorf("擦除笔画标识无效，请撤销后重试")
		}
		*total += len(c.Ranges) + 1
		if *total > 2000000 {
			return fmt.Errorf("擦除笔画过多，请拆分文档")
		}
		for _, v := range c.Ranges {
			if !finite(v[0]) || !finite(v[1]) || v[0] < 0 || v[1] <= v[0] || v[1] > 2000000 {
				return fmt.Errorf("擦除笔画范围无效，请撤销后重试")
			}
		}
	}
	return nil
}
