package engine

import (
	"strconv"
	"unicode/utf16"
)

// RebaseBodyCuts preserves untouched source characters across a text edit.
func RebaseBodyCuts(erasures []PageErasure, before, after string) []PageErasure {
	if before == after {
		return erasures
	}
	a, b := utf16.Encode([]rune(before)), utf16.Encode([]rune(after))
	start, end, nextEnd := 0, len(a), len(b)
	for start < end && start < nextEnd && a[start] == b[start] {
		start++
	}
	for end > start && nextEnd > start && a[end-1] == b[nextEnd-1] {
		end--
		nextEnd--
	}
	out := append([]PageErasure(nil), erasures...)
	for i, e := range out {
		if e.Cuts == nil {
			continue
		}
		cuts := []InkCut{}
		for _, c := range *e.Cuts {
			if c.ObjectID == "" {
				n, err := strconv.Atoi(c.Part)
				if err != nil || n >= start && n < end {
					continue
				}
				if n >= end {
					c.Part = strconv.Itoa(n + nextEnd - end)
				}
			}
			cuts = append(cuts, c)
		}
		out[i].Cuts = &cuts
	}
	return out
}
