package engine

import "fmt"

type WritingSettings struct {
	OffsetX    float64  `json:"offsetX"`
	OffsetY    float64  `json:"offsetY"`
	PenUp      *float64 `json:"penUp"`
	PenDown    *float64 `json:"penDown"`
	OriginMode string   `json:"originMode"`
	Sound      bool     `json:"sound"`
}

func DefaultWriting() WritingSettings {
	up, down := 7.0, 3.4
	return WritingSettings{PenUp: &up, PenDown: &down, OriginMode: "auto", Sound: true}
}

func (w WritingSettings) Validate() error {
	if w.PenUp == nil || w.PenDown == nil {
		return fmt.Errorf("请在书写参数中填写抬笔位置和落笔位置")
	}
	if !finite(w.OffsetX) || !finite(w.OffsetY) || !finite(*w.PenUp) || !finite(*w.PenDown) {
		return fmt.Errorf("书写参数必须为有效数字")
	}
	if w.OffsetX < -BedSize || w.OffsetX > BedSize || w.OffsetY < -BedSize || w.OffsetY > BedSize {
		return fmt.Errorf("XY 偏移须在 -256 至 256 mm 之间")
	}
	if *w.PenDown < 0 || *w.PenUp > BedSize || *w.PenUp-*w.PenDown < 0.1 {
		return fmt.Errorf("抬笔须高于落笔至少 0.1 mm，Z 位置须在 0–256 mm 内")
	}
	if w.OriginMode != "manual" && w.OriginMode != "auto" {
		return fmt.Errorf("请选择有效原点模式")
	}
	return nil
}
