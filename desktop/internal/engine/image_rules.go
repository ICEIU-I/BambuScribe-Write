package engine

import "math"

type imageRule struct {
	horizontal       bool
	axis, start, end float64
}

// Detect ruled tables before masking OCR text. Each candidate needs sustained
// ink support along its length; separate cells and intentional open edges are
// never inferred from the outside bounding rectangle.
func rectifiedTableLines(mask []bool, width, height int) []Stroke {
	var rules []imageRule
	for _, horizontal := range []bool{true, false} {
		major, minor := width, height
		if !horizontal {
			major, minor = height, width
		}
		at := func(x, y int) bool {
			if horizontal {
				return mask[y*width+x]
			}
			return mask[x*width+y]
		}
		minimum := max(30, major/5)
		var candidates []imageRule
		for y := 0; y < minor; y++ {
			first, last, hits := -1, -1, 0
			flush := func() {
				if first >= 0 && last-first+1 >= minimum && float64(hits)/float64(last-first+1) > .88 {
					candidates = append(candidates, imageRule{horizontal, float64(y), float64(first), float64(last)})
				}
				first, last, hits = -1, -1, 0
			}
			for x := 0; x < major; x++ {
				ink := false
				for yy := max(0, y-2); yy <= min(minor-1, y+2); yy++ {
					if at(x, yy) {
						ink = true
						break
					}
				}
				if ink {
					if first < 0 {
						first = x
					}
					last = x
					hits++
				} else if last >= 0 && x-last > 4 {
					flush()
				}
			}
			flush()
		}
		// Merge the bands belonging to the same rule, retaining their supported span.
		used := make([]bool, len(candidates))
		for i, c := range candidates {
			if used[i] {
				continue
			}
			used[i] = true
			low, high := c.axis, c.axis
			start, end := c.start, c.end
			for j := i + 1; j < len(candidates); j++ {
				d := candidates[j]
				if d.axis > high+1.1 {
					break
				}
				overlap := math.Min(end, d.end) - math.Max(start, d.start)
				if !used[j] && overlap > .75*math.Min(end-start, d.end-d.start) && d.axis-low <= 9 {
					used[j] = true
					high = d.axis
					start = math.Min(start, d.start)
					end = math.Max(end, d.end)
				}
			}
			// Refine the centre using only supporting pixels, not OCR boxes or text.
			axis := (low + high) / 2
			sum, n := 0., 0.
			for x := int(start); x <= int(end); x++ {
				for y := max(0, int(axis)-2); y <= min(minor-1, int(axis)+2); y++ {
					if at(x, y) {
						sum += float64(y)
						n++
					}
				}
			}
			if n > 0 {
				axis = sum / n
			}
			rules = append(rules, imageRule{horizontal, axis, start, end})
		}
	}
	h, v := 0, 0
	for _, r := range rules {
		if r.horizontal {
			h++
		} else {
			v++
		}
	}
	if h < 3 || v < 2 {
		return nil
	} // axes/curves are handled by the connected-path fitter
	// Nearby rule ends meet exactly. Long blank spans (merged cells) stay blank.
	for i := range rules {
		for j := range rules {
			if rules[i].horizontal == rules[j].horizontal {
				continue
			}
			a, b := &rules[i], rules[j]
			if a.axis < b.start-5 || a.axis > b.end+5 {
				continue
			}
			if math.Abs(a.start-b.axis) <= 5 {
				a.start = b.axis
			}
			if math.Abs(a.end-b.axis) <= 5 {
				a.end = b.axis
			}
		}
	}
	strokes := make([]Stroke, 0, len(rules))
	for _, r := range rules {
		if r.horizontal {
			strokes = append(strokes, Stroke{{r.start, r.axis}, {r.end, r.axis}})
		} else {
			strokes = append(strokes, Stroke{{r.axis, r.start}, {r.axis, r.end}})
		}
		// Remove the original near-axis ink so a second traced copy cannot remain,
		// but leave the positions where another line crosses. Cutting there would
		// split that line with nothing left to rejoin it: a wire crossing a rule
		// lost 2.9mm and every bar of a ground symbol was destroyed this way.
		for m := max(0, int(r.start)-1); m <= int(r.end)+1; m++ {
			if crossesImageRule(mask, width, height, r, m) {
				continue
			}
			for a := int(r.axis) - 4; a <= int(r.axis)+4; a++ {
				x, y := m, a
				if !r.horizontal {
					x, y = a, m
				}
				if x >= 0 && x < width && y >= 0 && y < height {
					mask[y*width+x] = false
				}
			}
		}
	}
	return strokes
}

// inkInWindow reports ink anywhere in the square window around (x, y).
func inkInWindow(mask []bool, width, height, x, y, radius int) bool {
	for py := max(0, y-radius); py <= min(height-1, y+radius); py++ {
		for px := max(0, x-radius); px <= min(width-1, x+radius); px++ {
			if mask[py*width+px] {
				return true
			}
		}
	}
	return false
}

// crossesImageRule reports a line passing through r at position m: ink just
// outside the deletion band on both sides. The caller keeps the band's ink
// there, so the crossing line stays in one piece. Judged against the mask as
// it stands now, so rules that cross each other are still deleted in full.
//
// A line running alongside would put ink on one side only, so both sides are
// required. Three near-parallel lines within a few pixels of each other could
// still read as a crossing and leave the middle one's ink in place, drawn twice
// — a cheaper mistake than the holes this replaces, which cut wires and erased
// every bar of a ground symbol.
func crossesImageRule(mask []bool, width, height int, r imageRule, m int) bool {
	axis := int(math.Round(r.axis))
	before := func(offset int) bool {
		if r.horizontal {
			return inkInWindow(mask, width, height, m, axis+offset, 1)
		}
		return inkInWindow(mask, width, height, axis+offset, m, 1)
	}
	return before(-5) && before(5)
}

// Rule ends and the lines they meet are traced separately, so a box corner or a
// stub joining a wire stops a few pixels short of the rule axis. Extend those
// ends onto the axis; the pen then turns exactly where the two lines meet.
func snapStrokesToRules(strokes []Stroke, rules []Stroke) []Stroke {
	// Wide enough to cross the deletion band and the few pixels thinning eats
	// from the tip beyond it; a line ending this near a rule is meeting it.
	const reach = 10.
	out := make([]Stroke, len(strokes))
	for i, source := range strokes {
		s := append(Stroke(nil), source...)
		out[i] = s
		if len(s) < 2 {
			continue
		}
		for end := 0; end < 2; end++ {
			idx := 0
			if end == 1 {
				idx = len(s) - 1
			}
			p := s[idx]
			// The local tangent, not the chord: a long curve can end near a rule
			// while pointing somewhere else entirely.
			dir := endpointDirection(s, end)
			if dir.X == 0 && dir.Y == 0 {
				continue
			}
			best, hit, found := reach, Point{}, false
			for _, rule := range rules {
				if len(rule) != 2 {
					continue
				}
				a, b := rule[0], rule[1]
				var axis, lo, hi, gap float64
				if a.Y == b.Y {
					if math.Abs(dir.Y) < 1e-6 {
						continue
					}
					axis, lo, hi = a.Y, math.Min(a.X, b.X), math.Max(a.X, b.X)
					gap = (axis - p.Y) / dir.Y
				} else {
					if math.Abs(dir.X) < 1e-6 {
						continue
					}
					axis, lo, hi = a.X, math.Min(a.Y, b.Y), math.Max(a.Y, b.Y)
					gap = (axis - p.X) / dir.X
				}
				// Only reach forward, and never past another rule's slack.
				if gap <= 0 || gap >= best {
					continue
				}
				point := Point{p.X + dir.X*gap, p.Y + dir.Y*gap}
				along := point.X
				if a.X == b.X {
					along = point.Y
				}
				if along < lo-2 || along > hi+2 {
					continue
				}
				best, hit, found = gap, point, true
			}
			if found {
				s[idx] = hit
			}
		}
	}
	return out
}
