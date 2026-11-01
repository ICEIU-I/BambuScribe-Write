package engine

import "sort"

// Reconstruct chains of forward-facing gaps, including along bends. Text has
// already been removed; solids and closed symbols cannot become dashed lines.
func normalizeDashedStrokes(strokes []Stroke) []Stroke {
	const maxGap = 36.
	links := []strokeLink{}
	lengths := make([]float64, len(strokes))
	bins := map[[2]int][]strokeEnd{}
	for i, s := range strokes {
		lengths[i] = strokeLength(s)
		if lengths[i] < 1.2 || lengths[i] > 55 || distance(s[0], s[len(s)-1]) < lengths[i]*.65 {
			continue
		}
		for end := 0; end < 2; end++ {
			p := endpoint(s, end)
			key := [2]int{int(p.X / maxGap), int(p.Y / maxGap)}
			bins[key] = append(bins[key], strokeEnd{i, end})
		}
	}
	for i, a := range strokes {
		if lengths[i] < 1.2 || lengths[i] > 55 || distance(a[0], a[len(a)-1]) < lengths[i]*.65 {
			continue
		}
		for ae := 0; ae < 2; ae++ {
			p := endpoint(a, ae)
			key := [2]int{int(p.X / maxGap), int(p.Y / maxGap)}
			for by := key[1] - 1; by <= key[1]+1; by++ {
				for bx := key[0] - 1; bx <= key[0]+1; bx++ {
					for _, other := range bins[[2]int{bx, by}] {
						j, be := other.index, other.end
						if j <= i {
							continue
						}
						b := strokes[j]
						q := endpoint(b, be)
						gap := distance(p, q)
						if gap < .8 || gap > maxGap {
							continue
						}
						dir := Point{(q.X - p.X) / gap, (q.Y - p.Y) / gap}
						u, v := endpointDirection(a, ae), endpointDirection(b, be)
						d1, d2 := u.X*dir.X+u.Y*dir.Y, -v.X*dir.X-v.Y*dir.Y
						if d1 < .78 || d2 < .78 {
							continue
						}
						links = append(links, strokeLink{i, j, ae, be, gap + 12*(2-d1-d2)})
					}
				}
			}
		}
	}
	out := []Stroke{}
	for _, chain := range connectImageStrokes(strokes, links) {
		if len(chain.ids) >= 5 && strokeLength(chain.path) > 40 {
			parts := make([]float64, len(chain.ids))
			for i, id := range chain.ids {
				parts[i] = lengths[id]
			}
			sort.Float64s(parts)
			sort.Float64s(chain.gaps)
			ink := max(5., min(12., parts[len(parts)/2]))
			gap := max(2., min(ink*.9, chain.gaps[len(chain.gaps)/2]))
			out = append(out, dashImageStroke(smoothImageStroke(resampleImageStroke(chain.path, 2)), ink, gap)...)
		} else {
			for _, id := range chain.ids {
				out = append(out, strokes[id])
			}
		}
	}
	return out
}
func resampleImageStroke(path Stroke, step float64) Stroke {
	if len(path) == 0 || step <= 0 {
		return path
	}
	out := Stroke{path[0]}
	remaining := step
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		length := distance(a, b)
		for length >= remaining && length > 1e-8 {
			a = mixPoint(a, b, remaining/length)
			out = append(out, a)
			length -= remaining
			remaining = step
		}
		remaining -= length
	}
	if distance(out[len(out)-1], path[len(path)-1]) > 1e-8 {
		out = append(out, path[len(path)-1])
	}
	return out
}
