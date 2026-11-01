package tableimport

import (
	"bambuscribe/desktop/internal/engine"
	"fmt"
	"math"
)

func validatePositions(positions []Position, count int, s engine.LayoutSettings) error {
	if len(positions) == 0 {
		return nil
	}
	if len(positions) != count {
		return fmt.Errorf("表格位置数量不匹配，请重新预览")
	}
	for i, p := range positions {
		if p.Page < 1 || p.Page > 20001 || !finitePosition(p.X) || !finitePosition(p.Y) || !finitePosition(p.Width) || p.Width < 3 || p.X < s.MarginLeft-1e-8 || p.Y < s.MarginTop-1e-8 || p.X+p.Width > s.PaperWidth-s.MarginRight+1e-8 {
			return fmt.Errorf("表格 %d 的位置或宽度超出纸面可用范围，请拖回页内", i+1)
		}
	}
	return nil
}

func finitePosition(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func placeImportGroup(index int, p Position, height float64, s engine.LayoutSettings, occupied map[int][]rect, manual bool) (Position, error) {
	bottom := s.PaperHeight - s.MarginBottom
	for {
		if p.Page > 20001 {
			return p, fmt.Errorf("页面数量过多，请拆分文档")
		}
		if p.Y+height > bottom+1e-8 {
			if manual {
				return p, fmt.Errorf("表格 %d 超出本页底部，请向上移动或换页", index+1)
			}
			p.Page++
			p.Y = s.MarginTop
			continue
		}
		// Manual positions may overlap objects. Body text is reflowed by the
		// shared engine; automatic placement still looks for an empty area.
		if manual {
			return p, nil
		}
		candidate := rect{p.X, p.Y, p.Width, height}
		next := p.Y
		for _, r := range occupied[p.Page-1] {
			if overlap(candidate, r, 2) {
				next = max(next, r.y+r.h+3)
			}
		}
		if next == p.Y {
			return p, nil
		}
		p.Y = next
	}
}
