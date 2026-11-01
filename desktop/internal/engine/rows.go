package engine

import (
	"crypto/sha256"
	"fmt"
	"math"
)

func (s LayoutSettings) rows(page int) (int, float64) {
	count := s.LineCount
	if s.LineMode == "random" {
		// Stable per-page variation keeps typing, reload and export on the same grid.
		h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", s.TemplateName, page)))
		count += int(h[0]%3) - 1
	}
	count = max(1, count)
	if page < len(s.PageLayouts) {
		count = s.PageLayouts[page].LineCount
	}
	height := s.PaperHeight - s.MarginTop - s.MarginBottom
	pitch := height / float64(count) * s.LineSpacing / 100
	return max(1, int(math.Floor(height/pitch+1e-8))), pitch
}

func (s LayoutSettings) normalized() LayoutSettings {
	// Older projects have no spacing percentages. Keep their original grid.
	if s.LetterSpacing == 0 {
		s.LetterSpacing = 100
	}
	if s.LineSpacing == 0 {
		s.LineSpacing = 100
	}
	if s.LineMode == "" {
		s.LineMode = "fixed"
		if s.LineHeight > 0 {
			s.LineCount = max(1, int((s.PaperHeight-s.MarginTop-s.MarginBottom)/s.LineHeight))
		}
		if s.LineCount == 0 {
			s.LineCount = 16
		}
	}
	if s.TemplateName == "" {
		s.TemplateName = "新建笔记"
	}
	s.LineHeight = (s.PaperHeight - s.MarginTop - s.MarginBottom) / float64(s.LineCount)
	return s
}
