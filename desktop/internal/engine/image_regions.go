package engine

import (
	"fmt"
	"unicode/utf8"
)

func validateImageRegion(r ImageTextRegion) error {
	valid := func(x, y, w, h float64) bool {
		return finite(x) && finite(y) && finite(w) && finite(h) && x >= -.001 && y >= -.001 && w > 0 && h > 0 && x+w <= 1.001 && y+h <= 1.001
	}
	if r.ID == "" || !valid(r.X, r.Y, r.Width, r.Height) {
		return fmt.Errorf("文字框超出图片，请移动或缩小文字框")
	}
	if r.SourceBounds != nil {
		b := r.SourceBounds
		if !valid(b.X, b.Y, b.Width, b.Height) {
			return fmt.Errorf("原文字区域无效，请恢复文字框")
		}
	}
	if len(r.ErasePath) > 0 {
		if r.Mode != "erase" || r.Source != "manual" || len(r.ErasePath) < 2 {
			return fmt.Errorf("自由擦除轨迹无效")
		}
		for _, p := range r.ErasePath {
			if !finite(p.X) || !finite(p.Y) || p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
				return fmt.Errorf("自由擦除轨迹超出图片范围")
			}
		}
	}
	if r.EraseRadius != 0 && (!finite(r.EraseRadius) || r.EraseRadius < 0.002 || r.EraseRadius > 0.15) {
		return fmt.Errorf("自由擦除笔刷粗细无效")
	}
	if utf8.RuneCountInString(r.Text) > 500 {
		return fmt.Errorf("图片文字最多 500 字，请拆分文字框")
	}
	if r.Mode != "" && r.Mode != "handwriting" && r.Mode != "erase" && r.Mode != "keep" {
		return fmt.Errorf("文字处理方式无效")
	}
	return nil
}
