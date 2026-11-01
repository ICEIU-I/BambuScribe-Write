package engine

import (
	"math"
	"sort"
)

// Ellipsis dots are often missed by OCR and become isolated pixels after
// thinning. Protect repeated, evenly spaced dot groups before speck filtering.
// A single isolated speck remains eligible for removal.
func extractImageDotGroups(mask []bool, width, height int) []bool {
	type dot struct {
		x, y, size float64
		pixels     []int
	}
	dots := []dot{}
	for _, pixels := range smallTextComponents(mask, width, height) {
		first := pixels[0]
		left, right, top, bottom := first%width, first%width, first/width, first/width
		for _, p := range pixels {
			x, y := p%width, p/width
			left, right = min(left, x), max(right, x)
			top, bottom = min(top, y), max(bottom, y)
		}
		w, h := right-left+1, bottom-top+1
		if w <= 8 && h <= 8 && max(w, h) <= 2*min(w, h) {
			dots = append(dots, dot{float64(left+right) / 2, float64(top+bottom) / 2, float64(max(w, h)), pixels})
		}
	}
	sort.Slice(dots, func(i, j int) bool { return dots[i].x < dots[j].x })
	rows := map[int][]int{}
	for i, d := range dots {
		row := int(d.y / 4)
		for y := row - 1; y <= row+1; y++ {
			rows[y] = append(rows[y], i)
		}
	}
	kept := make([]bool, len(mask))
	for i, a := range dots {
		line := rows[int(a.y/4)]
		for j := sort.SearchInts(line, i+1); j < len(line); j++ {
			b := dots[line[j]]
			gap := b.x - a.x
			if gap > 80 {
				break
			}
			if gap < 3 || gap < 2*max(a.size, b.size) || math.Abs(a.y-b.y) > 2 || max(a.size, b.size) > 2*min(a.size, b.size) {
				continue
			}
			for k := j + 1; k < len(line); k++ {
				c := dots[line[k]]
				if c.x-b.x > gap*1.3 {
					break
				}
				if c.x-b.x < gap*.7 || math.Abs(c.y-a.y) > 2 || max(a.size, c.size) > 2*min(a.size, c.size) {
					continue
				}
				for _, d := range []dot{a, b, c} {
					for _, p := range d.pixels {
						kept[p] = true
					}
				}
				break
			}
		}
	}
	for i, keep := range kept {
		if keep {
			mask[i] = false
		}
	}
	return kept
}
