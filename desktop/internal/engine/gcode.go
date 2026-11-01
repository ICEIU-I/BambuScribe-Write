package engine

import (
	"fmt"
	"strings"
)

const BedSize = 256.0

func GenerateGCode(page Page, s LayoutSettings, w WritingSettings) (string, error) {
	if err := w.Validate(); err != nil {
		return "", err
	}
	if err := s.Validate(); err != nil {
		return "", err
	}
	if len(page.Missing) > 0 {
		return "", fmt.Errorf("第 %d 页有缺字，请先修改文字", page.Number)
	}
	if len(page.Issues) > 0 {
		return "", fmt.Errorf("第 %d 页：%s", page.Number, page.Issues[0])
	}
	if len(page.Strokes) == 0 {
		return "", fmt.Errorf("第 %d 页没有可书写笔画", page.Number)
	}
	if len(page.StrokeKinds) != 0 && len(page.StrokeKinds) != len(page.Strokes) {
		return "", fmt.Errorf("笔画类型不完整，请重新排版后导出")
	}
	parkX, parkY, err := parkingTarget(s, w)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "; A1 handwriting / page %d\n; Home and position once before the first page; replace paper between pages.\nM400\nG21\nM104 S0\nM140 S0\nM106 S0\nM17\nG90\n", page.Number)
	b.WriteString(a1MotionSetup)
	if w.OriginMode == "auto" {
		fmt.Fprintf(&b, "G92 X0 Y%.3f\n", s.PaperHeight)
	}
	fmt.Fprintf(&b, "G1 Z%.3f F%d\nM400\n", *w.PenUp, motionFeedRate)
	// Auto's start and park are the same physical position. Do not include
	// coordinate Y0: its distance depends on the paper height, not the ink.
	minX, minY, maxX, maxY := parkX, parkY, parkX, parkY
	// Chixie uses G1 for both writing and raised-pen repositioning. Keep the
	// same motion command for the stroke start, stroke points and final park.
	move := func(x, y int64, feed int) {
		fmt.Fprintf(&b, "G1 X%.3f Y%.3f F%d\n", float64(x)/1000, float64(y)/1000, feed)
	}
	for si, stroke := range page.Strokes {
		if len(stroke) < 2 {
			return "", fmt.Errorf("笔画数据不完整")
		}
		for i, p := range stroke {
			x, y, err := machineTarget(p, s, w)
			if err != nil {
				return "", err
			}
			minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
			if maxX-minX > 256000 || maxY-minY > 256000 {
				return "", fmt.Errorf("书写范围超过 A1 行程，请缩小页面或字号")
			}
			feed := motionFeedRate
			if i > 0 && len(page.StrokeKinds) > 0 && page.StrokeKinds[si] == strokeKindTableGrid {
				feed = tableGridFeedRate
			}
			move(x, y, feed)
			if i == 0 {
				fmt.Fprintf(&b, "G1 Z%.3f F%d\n", *w.PenDown, motionFeedRate)
			}
		}
		fmt.Fprintf(&b, "G1 Z%.3f F%d\n", *w.PenUp, motionFeedRate)
	}
	// Clear the finished sheet before the long return, then keep that height
	// while the operator replaces the paper. Never lower a higher pen-up Z.
	fmt.Fprintf(&b, "G1 Z%.3f F%d\nM400\n", max(*w.PenUp, pageReturnHeight), motionFeedRate)
	move(parkX, parkY, motionFeedRate)
	b.WriteString("M400\n")
	if w.Sound {
		b.WriteString(completionSoundGCode)
	}
	// Preserve the parked XY position and calibrated Z for the next page.
	// The job finishes here; the operator replaces the paper and starts it.
	b.WriteString("M400\n")
	return b.String(), nil
}
