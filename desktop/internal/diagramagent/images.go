package diagramagent

import (
	"bambuscribe/desktop/internal/tableimport"
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"image/png"
	"strings"
)

func prepareImages(data string) ([]string, int, int, error) {
	if err := tableimport.ValidateImage(data); err != nil {
		return nil, 0, 0, err
	}
	_, encoded, _ := strings.Cut(data, ",")
	raw, _ := base64.StdEncoding.DecodeString(encoded)
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(raw))
	if cfg.Width < 2 || cfg.Height < 2 || cfg.Width > 8192 || cfg.Height > 8192 {
		return nil, 0, 0, fmt.Errorf("图片边长须为 2–8192 像素，请裁剪后重试")
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("图片无法读取，请重新选择")
	}
	w, h := cfg.Width, cfg.Height
	cut := w / 2
	pictures := []string{data}
	for _, r := range []image.Rectangle{image.Rect(0, 0, min(w, cut+30), h), image.Rect(max(0, cut-30), 0, w, h)} {
		width, height := r.Dx()*2, r.Dy()*2
		if max(width, height) > 2048 {
			factor := 2048. / float64(max(width, height))
			width = max(1, int(float64(width)*factor))
			height = max(1, int(float64(height)*factor))
		}
		out := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(out, out.Bounds(), source, r, draw.Src, nil)
		var b bytes.Buffer
		if err := png.Encode(&b, out); err != nil {
			return nil, 0, 0, err
		}
		pictures = append(pictures, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(b.Bytes()))
	}
	return pictures, w, h, nil
}

func imagePrompt(w, h int, instruction string) string {
	cut := w / 2
	prompt := fmt.Sprintf("Reconstruct the entire %d x %d source image. The first image is the full source. The second image is an enlarged left crop, original extent (0,0)-(%d,%d). The third is an enlarged right crop, original extent (%d,0)-(%d,%d). All output coordinates must refer to the FIRST image. Double-check every printed label, all pin numbers, the number of circular contacts, and physical connectivity of wires where present.", w, h, min(w, cut+30), h, max(0, cut-30), w, h)
	if instruction != "" {
		prompt += "\n用户补充要求：" + instruction
	}
	return prompt
}
