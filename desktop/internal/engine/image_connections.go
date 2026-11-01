package engine

import (
	"math"
	"sort"
)

type strokeLink struct {
	a, b, ae, be int
	cost         float64
}
type strokeEnd struct{ index, end int }
type imageChain struct {
	path Stroke
	ids  []int
	gaps []float64
}

func endpoint(s Stroke, end int) Point {
	if end == 0 {
		return s[0]
	}
	return s[len(s)-1]
}
func endpointDirection(s Stroke, end int) Point {
	p := endpoint(s, end)
	q := p
	for i := 1; i < len(s); i++ {
		idx := i
		if end == 1 {
			idx = len(s) - 1 - i
		}
		q = s[idx]
		if distance(p, q) >= 3 {
			break
		}
	}
	length := distance(p, q)
	if length == 0 {
		return Point{}
	}
	return Point{(p.X - q.X) / length, (p.Y - q.Y) / length}
}
func orientedStroke(s Stroke, from int) Stroke {
	if from == 0 {
		return s
	}
	out := make(Stroke, len(s))
	for i, p := range s {
		out[len(s)-1-i] = p
	}
	return out
}
func connectImageStrokes(strokes []Stroke, links []strokeLink) []imageChain {
	sort.SliceStable(links, func(i, j int) bool { return links[i].cost < links[j].cost })
	n := len(strokes)
	connections := make([][2]strokeEnd, n)
	parent := make([]int, n)
	for i := range connections {
		connections[i] = [2]strokeEnd{{-1, 0}, {-1, 0}}
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for _, l := range links {
		ra, rb := root(l.a), root(l.b)
		if connections[l.a][l.ae].index >= 0 || connections[l.b][l.be].index >= 0 || ra == rb {
			continue
		}
		connections[l.a][l.ae] = strokeEnd{l.b, l.be}
		connections[l.b][l.be] = strokeEnd{l.a, l.ae}
		parent[rb] = ra
	}
	used := make([]bool, n)
	out := []imageChain{}
	for start := 0; start < n; start++ {
		if used[start] || connections[start][0].index >= 0 && connections[start][1].index >= 0 {
			continue
		}
		from := 0
		if connections[start][0].index >= 0 {
			from = 1
		}
		chain := imageChain{}
		for cur := start; cur >= 0; {
			used[cur] = true
			chain.ids = append(chain.ids, cur)
			s := orientedStroke(strokes[cur], from)
			if len(chain.path) > 0 {
				gap := distance(chain.path[len(chain.path)-1], s[0])
				chain.gaps = append(chain.gaps, gap)
				if gap < 1e-8 {
					s = s[1:]
				}
			}
			chain.path = append(chain.path, s...)
			next := connections[cur][1-from]
			cur, from = next.index, next.end
		}
		out = append(out, chain)
	}
	return out
}

// Branches of one junction do not all end on the same pixel after thinning, so
// endpoints match within a tolerance instead of exactly. A tolerance of 2px is
// under the gap any real pen-lift leaves and well over the 1-2px spread seen at
// a junction; wider matches must be near-collinear to stay unambiguous.
const junctionTolerance = 2.

// Shared endpoints are indexed: no all-pairs scan for each junction merge.
func joinImageJunctions(strokes []Stroke) []Stroke {
	ends := map[[2]int][]strokeEnd{}
	key := func(p Point) [2]int {
		return [2]int{int(math.Floor(p.X / junctionTolerance)), int(math.Floor(p.Y / junctionTolerance))}
	}
	for i, s := range strokes {
		for end := 0; end < 2; end++ {
			k := key(endpoint(s, end))
			ends[k] = append(ends[k], strokeEnd{i, end})
		}
	}
	links := []strokeLink{}
	// Iterate source order for deterministic tie-breaking, not map order.
	for i, s := range strokes {
		for ae := 0; ae < 2; ae++ {
			p := endpoint(s, ae)
			k := key(p)
			for by := k[1] - 1; by <= k[1]+1; by++ {
				for bx := k[0] - 1; bx <= k[0]+1; bx++ {
					for _, other := range ends[[2]int{bx, by}] {
						if other.index <= i {
							continue
						}
						gap := distance(p, endpoint(strokes[other.index], other.end))
						if gap > junctionTolerance {
							continue
						}
						u, v := endpointDirection(s, ae), endpointDirection(strokes[other.index], other.end)
						dot := u.X*v.X + u.Y*v.Y
						limit := -.85
						if gap > .5 {
							limit = -.94
						}
						if dot < limit {
							links = append(links, strokeLink{i, other.index, ae, other.end, dot})
						}
					}
				}
			}
		}
	}
	out := []Stroke{}
	for _, c := range connectImageStrokes(strokes, links) {
		out = append(out, c.path)
	}
	return out
}
