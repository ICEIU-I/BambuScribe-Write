package editor

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/project"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"image"
	"io"
	"math"
	"os"
	"strings"
)

// ReadFile bounds data before allocating a large JSON/image payload.
func ReadFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("文件过大：%s", path)
	}
	return b, nil
}
func imageData(path string) (string, int, int, error) {
	b, err := ReadFile(path, 9<<20)
	if err != nil {
		return "", 0, 0, err
	}
	c, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || (format != "png" && format != "jpeg") {
		return "", 0, 0, fmt.Errorf("仅支持 PNG 或 JPEG 图片")
	}
	return "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(b), c.Width, c.Height, nil
}
func imageIndex(p *project.Project, id string) (int, error) {
	for i, t := range p.Images {
		if t.ID == id {
			return i, nil
		}
	}
	return 0, fmt.Errorf("找不到图片 %q", id)
}

func editImage(p *project.Project, op Operation) (string, error) {
	if op.Tool == "image.insert" {
		a := struct {
			ID       string  `json:"id"`
			File     string  `json:"file"`
			Page     int     `json:"page"`
			X        float64 `json:"x"`
			Y        float64 `json:"y"`
			Width    float64 `json:"width"`
			Height   float64 `json:"height"`
			Rotation float64 `json:"rotation"`
		}{Page: 1, Width: 70}
		if err := Decode(op.Params, &a); err != nil {
			return "", err
		}
		if a.ID == "" {
			a.ID = "image-" + uuid.NewString()
		}
		if err := uniqueID(p, a.ID); err != nil {
			return "", err
		}
		data, w, h, err := imageData(a.File)
		if err != nil {
			return "", err
		}
		if a.Height == 0 {
			a.Height = a.Width * float64(h) / float64(w)
		}
		p.Images = append(p.Images, engine.ImageAsset{ID: a.ID, Page: a.Page - 1, Data: data, X: a.X, Y: a.Y, Width: a.Width, Height: a.Height, Rotation: a.Rotation, Mode: "line"})
		return a.ID, nil
	}
	var head struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(op.Params, &head); err != nil {
		return "", err
	}
	i, err := imageIndex(p, head.ID)
	if err != nil {
		return "", err
	}
	t := &p.Images[i]
	if op.Tool == "image.update" {
		var a struct {
			ID       string   `json:"id"`
			File     string   `json:"file"`
			Page     *int     `json:"page"`
			X        *float64 `json:"x"`
			Y        *float64 `json:"y"`
			Width    *float64 `json:"width"`
			Height   *float64 `json:"height"`
			Rotation *float64 `json:"rotation"`
		}
		if err := Decode(op.Params, &a); err != nil {
			return "", err
		}
		if a.File != "" {
			data, _, _, err := imageData(a.File)
			if err != nil {
				return "", err
			}
			t.Data = data
			// Manual corrections are retained; automatic OCR belongs to the new source.
			var manual []engine.ImageTextRegion
			for _, r := range t.TextRegions {
				if r.Source == "manual" {
					manual = append(manual, r)
				}
			}
			t.TextRegions = manual
		}
		if a.Page != nil {
			t.Page = *a.Page - 1
		}
		if a.X != nil {
			t.X = *a.X
		}
		if a.Y != nil {
			t.Y = *a.Y
		}
		if a.Width != nil {
			t.Width = *a.Width
		}
		if a.Height != nil {
			t.Height = *a.Height
		}
		if a.Rotation != nil {
			t.Rotation = *a.Rotation
		}
		return t.ID, nil
	}
	return editRegion(t, op)
}

// Region coordinates and brush radius are millimeters in the unrotated image.
// They are converted to the existing normalized, aspect-preserving storage.
func editRegion(t *engine.ImageAsset, op Operation) (string, error) {
	var a struct {
		ID       string         `json:"id"`
		RegionID string         `json:"regionId"`
		Text     *string        `json:"text"`
		X        *float64       `json:"x"`
		Y        *float64       `json:"y"`
		Width    *float64       `json:"width"`
		Height   *float64       `json:"height"`
		Points   []engine.Point `json:"points"`
		Radius   *float64       `json:"radius"`
	}
	if err := Decode(op.Params, &a); err != nil {
		return "", err
	}
	update := op.Tool == "image.text.update"
	remove := strings.HasSuffix(op.Tool, ".delete")
	i := -1
	for j, r := range t.TextRegions {
		if r.ID == a.RegionID {
			i = j
			break
		}
	}
	if update || remove {
		if i < 0 {
			return "", fmt.Errorf("找不到图片内区域 %q", a.RegionID)
		}
		r := t.TextRegions[i]
		if r.Source != "manual" || (strings.Contains(op.Tool, ".erase.") != (r.Mode == "erase")) {
			return "", fmt.Errorf("只能修改对应类型的人工文字或擦除区域")
		}
		if remove {
			t.TextRegions = append(t.TextRegions[:i], t.TextRegions[i+1:]...)
			return a.RegionID, nil
		}
	} else {
		if a.RegionID == "" {
			a.RegionID = "region-" + uuid.NewString()
		}
		if i >= 0 {
			return "", fmt.Errorf("区域 ID 已存在")
		}
	}
	r := engine.ImageTextRegion{ID: a.RegionID, Source: "manual", Mode: "handwriting"}
	if update {
		r = t.TextRegions[i]
	}
	if a.Text != nil {
		r.Text = *a.Text
	}
	if a.X != nil {
		r.X = *a.X / t.Width
	}
	if a.Y != nil {
		r.Y = *a.Y / t.Height
	}
	if a.Width != nil {
		r.Width = *a.Width / t.Width
	}
	if a.Height != nil {
		r.Height = *a.Height / t.Height
	}
	if op.Tool == "image.erase.rect" {
		r.Mode = "erase"
	}
	if op.Tool == "image.erase.path" {
		if len(a.Points) == 0 {
			return "", fmt.Errorf("自由擦除需要至少一个轨迹点")
		}
		if len(a.Points) > 20000 {
			return "", fmt.Errorf("擦除轨迹最多 20,000 点")
		}
		r.Mode = "erase"
		r.EraseRadius = .018
		if a.Radius != nil {
			r.EraseRadius = *a.Radius / math.Min(t.Width, t.Height)
		}
		if r.EraseRadius < .002 || r.EraseRadius > .15 {
			return "", fmt.Errorf("笔刷半径须为图片短边的 0.2–15%%，单位 mm")
		}
		minX, minY, maxX, maxY := 1.0, 1.0, 0.0, 0.0
		for _, p := range a.Points {
			p.X /= t.Width
			p.Y /= t.Height
			if p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
				return "", fmt.Errorf("擦除轨迹超出图片范围")
			}
			r.ErasePath = append(r.ErasePath, p)
			minX, minY, maxX, maxY = math.Min(minX, p.X), math.Min(minY, p.Y), math.Max(maxX, p.X), math.Max(maxY, p.Y)
		}
		if len(r.ErasePath) == 1 {
			r.ErasePath = append(r.ErasePath, r.ErasePath[0])
		}
		r.X, r.Y = math.Min(minX, 1-1e-6), math.Min(minY, 1-1e-6)
		r.Width, r.Height = math.Max(1e-6, maxX-r.X), math.Max(1e-6, maxY-r.Y)
	}
	if update {
		t.TextRegions[i] = r
	} else {
		t.TextRegions = append(t.TextRegions, r)
	}
	return r.ID, nil
}
