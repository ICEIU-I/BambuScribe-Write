package engine

import (
	"bambuscribe/desktop/internal/diagram"
	"fmt"
	"strings"
)

// ValidateDiagram checks handwriting coverage locally, without another model
// request. An unsupported formula must not become a silently missing label.
func (f *Font) ValidateDiagram(scene *diagram.Scene) error {
	if err := scene.Validate(); err != nil {
		return err
	}
	asset := ImageAsset{Width: float64(scene.Width), Height: float64(scene.Height)}
	for _, l := range scene.Labels {
		box := diagramTextBox(l, scene)
		if len(imageRegionTextStrokes(f, box, asset, DefaultLayout(), scene.Width, scene.Height)) == 0 {
			return fmt.Errorf("图中文字或公式暂不支持：%s；请在补充要求中指定简写后重试", l.Text)
		}
	}
	return nil
}

func diagramTextBox(l diagram.Label, s *diagram.Scene) OCRBox {
	format := ""
	if strings.ContainsAny(l.Text, "_^") {
		format = "math"
	}
	return OCRBox{Text: l.Text, X: l.X / float64(s.Width), Y: l.Y / float64(s.Height), W: l.Width / float64(s.Width), H: l.Height / float64(s.Height), TextFormat: format}
}

func diagramImageStrokes(f *Font, asset ImageAsset, settings LayoutSettings) (imageRender, error) {
	scene := asset.Diagram
	if err := f.ValidateDiagram(scene); err != nil {
		return imageRender{}, err
	}
	var source []Stroke
	for _, path := range scene.Paths() {
		stroke := make(Stroke, len(path))
		for i, p := range path {
			stroke[i] = imagePoint(asset, asset.X+p.X/float64(scene.Width)*asset.Width, asset.Y+p.Y/float64(scene.Height)*asset.Height)
		}
		source = append(source, stroke)
	}
	for _, label := range scene.Labels {
		source = append(source, imageRegionTextStrokes(f, diagramTextBox(label, scene), asset, settings, scene.Width, scene.Height)...)
	}
	// Existing manual masks and text stay attached when a legacy image is
	// reconstructed. Saved automatic OCR is superseded by the complete scene.
	regions := []ImageTextRegion{}
	for _, r := range asset.TextRegions {
		if r.Source != "manual" {
			continue
		}
		if err := validateImageRegion(r); err != nil {
			return imageRender{}, err
		}
		r.Strokes = nil
		r.SourceStrokes = nil
		if r.Mode == "handwriting" || r.Mode == "" {
			box := OCRBox{Text: r.Text, X: r.X, Y: r.Y, W: r.Width, H: r.Height, TextFormat: r.TextFormat}
			r.Strokes = imageRegionTextStrokes(f, box, asset, settings, scene.Width, scene.Height)
			if len(r.Strokes) == 0 && strings.TrimSpace(r.Text) != "" {
				return imageRender{}, fmt.Errorf("文本框含有字库不支持的字符，请修改文字")
			}
		}
		regions = append(regions, r)
	}
	graphics := eraseImageStrokes(source, asset, regions)
	strokes := append([]Stroke{}, graphics...)
	for i := range regions {
		regions[i].SourceStrokes = regions[i].Strokes
		regions[i].Strokes = eraseImageStrokes(regions[i].Strokes, asset, regions[i+1:])
		strokes = append(strokes, regions[i].Strokes...)
	}
	warnings := []string{}
	for _, note := range scene.Uncertainties {
		warnings = append(warnings, "图像识别提示："+note)
	}
	return imageRender{Strokes: strokes, GraphicStrokes: graphics, SourceStrokes: source, TextRegions: regions, Warnings: warnings}, nil
}
