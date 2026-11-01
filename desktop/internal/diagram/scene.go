// Package diagram stores model-produced vector geometry in source pixels.
// It validates and samples primitives; it never infers or repairs connections.
package diagram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type Stroke struct {
	ID     string  `json:"id"`
	Role   string  `json:"role"`
	Points []Point `json:"points"`
	Dashed bool    `json:"dashed"`
}
type Circle struct {
	ID     string  `json:"id"`
	CX     float64 `json:"cx"`
	CY     float64 `json:"cy"`
	R      float64 `json:"r"`
	Dashed bool    `json:"dashed"`
}
type Label struct {
	Text   string  `json:"text"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
type Scene struct {
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	Strokes       []Stroke `json:"strokes"`
	Circles       []Circle `json:"circles"`
	Labels        []Label  `json:"labels"`
	Uncertainties []string `json:"uncertainties"`
}

func Decode(raw []byte) (*Scene, error) {
	if len(raw) > 1<<20 {
		return nil, fmt.Errorf("图像识别结果过大，请裁剪后重试")
	}
	var s Scene
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return nil, fmt.Errorf("图像识别格式无效，请重试")
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("图像识别含多余内容，请重试")
	}
	return &s, s.Validate()
}

func (s *Scene) Validate() error {
	if s == nil {
		return fmt.Errorf("缺少图像识别结果")
	}
	if s.Width < 2 || s.Height < 2 || s.Width > 8192 || s.Height > 8192 || int64(s.Width)*int64(s.Height) > 40_000_000 {
		return fmt.Errorf("图像识别尺寸无效，请裁剪后重试")
	}
	if len(s.Strokes)+len(s.Circles)+len(s.Labels) == 0 || len(s.Strokes) > 500 || len(s.Circles) > 500 || len(s.Labels) > 500 || len(s.Uncertainties) > 100 {
		return fmt.Errorf("图像识别内容为空或过多，请裁剪后重试")
	}
	finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
	point := func(x, y float64) bool {
		return finite(x) && finite(y) && x >= 0 && y >= 0 && x <= float64(s.Width) && y <= float64(s.Height)
	}
	ids := map[string]bool{}
	id := func(v string) bool {
		if v == "" || len(v) > 200 || ids[v] {
			return false
		}
		ids[v] = true
		return true
	}
	count := 0
	for _, line := range s.Strokes {
		count += len(line.Points)
		if !id(line.ID) || len(line.Points) < 2 || len(line.Points) > 500 || count > 20000 || len(line.Role) > 100 {
			return fmt.Errorf("图像线段无效，请重试")
		}
		for _, p := range line.Points {
			if !point(p.X, p.Y) {
				return fmt.Errorf("图像线段超出原图，请重试")
			}
		}
	}
	for _, c := range s.Circles {
		if !id(c.ID) || !finite(c.R) || c.R <= 0 || !point(c.CX-c.R, c.CY-c.R) || !point(c.CX+c.R, c.CY+c.R) {
			return fmt.Errorf("图像圆形坐标无效，请重试")
		}
	}
	for _, l := range s.Labels {
		if strings.TrimSpace(l.Text) == "" || utf8.RuneCountInString(l.Text) > 500 || !finite(l.Width) || !finite(l.Height) || l.Width <= 0 || l.Height <= 0 || !point(l.X, l.Y) || !point(l.X+l.Width, l.Y+l.Height) {
			return fmt.Errorf("图像文字位置无效，请重试")
		}
	}
	for _, note := range s.Uncertainties {
		if utf8.RuneCountInString(note) > 500 {
			return fmt.Errorf("图像提示过长，请重试")
		}
	}
	return nil
}
