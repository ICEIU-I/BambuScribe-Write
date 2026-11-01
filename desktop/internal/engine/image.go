package engine

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"
)

const maxImageData = 12 << 20

func decodeImageData(data string) (image.Image, error) {
	raw, err := imageBytes(data)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("仅支持 PNG 或 JPEG 图片：%w", err)
	}
	return img, nil
}

func imageBytes(data string) ([]byte, error) {
	if len(data) == 0 || len(data) > maxImageData {
		return nil, fmt.Errorf("图片大小须不超过 12 MB")
	}
	comma := strings.IndexByte(data, ',')
	if comma < 0 || !strings.Contains(strings.ToLower(data[:comma]), "base64") {
		return nil, fmt.Errorf("图片数据格式不正确")
	}
	raw, err := base64.StdEncoding.DecodeString(data[comma+1:])
	if err != nil {
		return nil, fmt.Errorf("图片无法读取：%w", err)
	}
	return raw, nil
}

func imageLuma(img image.Image, x, y int) float64 {
	r, g, b, a := img.At(x, y).RGBA()
	if a < 0x8000 {
		return 255
	}
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 257
}

func imageSample(img image.Image, x, y, width, height int) float64 {
	b := img.Bounds()
	px := b.Min.X + int((float64(x)+0.5)/float64(width)*float64(b.Dx()))
	py := b.Min.Y + int((float64(y)+0.5)/float64(height)*float64(b.Dy()))
	if px >= b.Max.X {
		px = b.Max.X - 1
	}
	if py >= b.Max.Y {
		py = b.Max.Y - 1
	}
	return imageLuma(img, px, py)
}

func imagePoint(asset ImageAsset, x, y float64) Point {
	if math.Abs(asset.Rotation) < 1e-9 {
		return Point{X: x, Y: y}
	}
	cx, cy := asset.X+asset.Width/2, asset.Y+asset.Height/2
	a := asset.Rotation * math.Pi / 180
	dx, dy := x-cx, y-cy
	return Point{X: cx + dx*math.Cos(a) - dy*math.Sin(a), Y: cy + dx*math.Sin(a) + dy*math.Cos(a)}
}

type imageRender struct {
	GraphicStrokes []Stroke
	SourceStrokes  []Stroke
	Strokes        []Stroke
	TextRegions    []ImageTextRegion
	Warnings       []string
}

func imageStrokes(f *Font, asset ImageAsset, settings LayoutSettings) (imageRender, error) {
	img, err := decodeImageData(asset.Data)
	if err != nil {
		return imageRender{}, err
	}
	if asset.Width < 2 || asset.Height < 2 || !finite(asset.X) || !finite(asset.Y) || !finite(asset.Width) || !finite(asset.Height) || !finite(asset.Rotation) {
		return imageRender{}, fmt.Errorf("图片尺寸或位置无效")
	}
	if asset.X < 0 || asset.Y < 0 || asset.X+asset.Width > settings.PaperWidth || asset.Y+asset.Height > settings.PaperHeight {
		return imageRender{}, fmt.Errorf("图片超出纸张范围，请缩小或移动图片")
	}
	if asset.Diagram != nil {
		return diagramImageStrokes(f, asset, settings)
	}
	bounds := img.Bounds()
	raw, _ := imageBytes(asset.Data)
	// Source resolution, rather than physical placement size, determines the
	// trace. Resizing must not change which lines/dashes survive.
	ratio := math.Min(1, 1600/float64(max(bounds.Dx(), bounds.Dy())))
	cols, rows := int(float64(bounds.Dx())*ratio), int(float64(bounds.Dy())*ratio)
	if cols < 2 || rows < 2 {
		return imageRender{}, fmt.Errorf("图片分辨率过低，无法提取线条")
	}
	dark := make([]bool, cols*rows)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			dark[y*cols+x] = imageSample(img, x, y, cols, rows) < 230
		}
	}
	// Character checks need the ink both ways: split on it as printed, where a
	// rule crossing a label would otherwise read as a break between characters,
	// and score against what is left once that rule is removed, since its band
	// is ink no glyph could cover.
	var original []bool
	if needsImageOCR(asset.TextRegions) {
		original = append([]bool{}, dark...)
	}
	// Native glyph verification uses a separate rule-free copy. Actual text
	// is extracted before diagram rules so letter stems are not rectified.
	var scoreInk []bool
	if needsImageOCR(asset.TextRegions) {
		scoreInk = append([]bool(nil), dark...)
		rectifiedTableLines(scoreInk, cols, rows)
	}
	warnings := []string{}
	textBoxes := []OCRBox{}
	keepBoxes := []OCRBox{}
	regions := append([]ImageTextRegion{}, asset.TextRegions...)
	if needsImageOCR(asset.TextRegions) {
		boxes, warning := recognizeLocalOrNative(raw, func() []OCRBox { return cachedImageOCR(raw, func() []OCRBox { return recognizeImageText(raw) }) })
		if warning != "" {
			warnings = append(warnings, warning)
		}
		regions = resolveImageOCR(f, asset.TextRegions, boxes, original, scoreInk, cols, rows, asset)
	}
	for i := range regions {
		region := &regions[i]
		region.Strokes = nil
		region.SourceStrokes = nil
		if err := validateImageRegion(*region); err != nil {
			return imageRender{}, err
		}
		if region.SourceBounds == nil && region.Source != "manual" {
			region.SourceBounds = &RegionBounds{region.X, region.Y, region.Width, region.Height}
		}
		mode := region.Mode
		if mode == "" {
			mode = "handwriting"
		}
		box := OCRBox{Text: region.Text, X: region.X, Y: region.Y, W: region.Width, H: region.Height, Confidence: region.Confidence, TextFormat: region.TextFormat}
		if mode == "keep" {
			keepBoxes = append(keepBoxes, box)
			continue
		}
		if mode == "handwriting" {
			replacement := imageRegionTextStrokes(f, box, asset, settings, cols, rows)
			if len(replacement) == 0 {
				if manualTextBox(*region) && strings.TrimSpace(region.Text) != "" {
					return imageRender{}, fmt.Errorf("文本框含有字库不支持的字符，请修改文字")
				}
				keepBoxes = append(keepBoxes, box)
				continue
			}
			region.Strokes = replacement
		}
		if mode == "erase" && region.Source == "manual" {
			continue
		}
		if region.SourceBounds != nil {
			b := region.SourceBounds
			textBoxes = append(textBoxes, OCRBox{X: b.X, Y: b.Y, W: b.Width, H: b.Height})
		}
	}
	eraseImageText(dark, cols, rows, textBoxes)
	// Retained formula/text ink must not enter the curve, dashed-line or speck
	// heuristics: they can join characters and remove decimal points/exponents.
	keptText := extractImageText(dark, cols, rows, keepBoxes)
	straight := rectifiedTableLines(dark, cols, rows)
	keptDots := extractImageDotGroups(dark, cols, rows)
	if keptText == nil {
		keptText = keptDots
	} else {
		for i, ink := range keptDots {
			keptText[i] = keptText[i] || ink
		}
	}
	dark = thinImageMask(dark, cols, rows)
	paths := traceImageSkeleton(dark, cols, rows, func(x, y int) Point { return Point{float64(x), float64(y)} })
	paths, circles := regularizeImageCircles(paths)
	paths = snapStrokesToRules(paths, straight)
	paths = joinImageJunctions(paths)
	for i, s := range paths {
		paths[i] = fitDiagramStroke(s)
	}
	// Dashes are rebuilt whether or not the figure also has straight rules: the
	// two have nothing to do with each other, and a ruled figure with a dashed
	// line in it was left as fragments.
	paths = normalizeDashedStrokes(paths)
	paths = dropIsolatedSpecks(paths, 3, 4)
	paths = append(paths, circles...)
	paths = append(paths, traceKeptImageText(keptText, cols, rows)...)
	paths = append(straight, paths...)
	strokes := make([]Stroke, 0, len(paths))
	for _, path := range paths {
		stroke := make(Stroke, len(path))
		for i, p := range path {
			stroke[i] = imagePoint(asset, asset.X+p.X/float64(cols-1)*asset.Width, asset.Y+p.Y/float64(rows-1)*asset.Height)
		}
		strokes = append(strokes, stroke)
	}
	if len(strokes) == 0 {
		for idx, active := range dark {
			if !active {
				continue
			}
			x, y := idx%cols, idx/cols
			p := Point{asset.X + float64(x)/float64(cols-1)*asset.Width, asset.Y + float64(y)/float64(rows-1)*asset.Height}
			q := Point{math.Min(asset.X+asset.Width, p.X+.02), p.Y}
			if q.X == p.X {
				q.X -= .02
			}
			strokes = append(strokes, Stroke{imagePoint(asset, p.X, p.Y), imagePoint(asset, q.X, q.Y)})
		}
	}
	sourceStrokes := strokes
	graphicStrokes := eraseImageStrokes(strokes, asset, regions)
	strokes = append([]Stroke{}, graphicStrokes...)
	for i := range regions {
		regions[i].SourceStrokes = regions[i].Strokes
		regions[i].Strokes = eraseImageStrokes(regions[i].Strokes, asset, regions[i+1:])
		strokes = append(strokes, regions[i].Strokes...)
	}
	if len(strokes) == 0 && len(regions) == 0 {
		return imageRender{}, fmt.Errorf("图片没有检测到可书写线条，请提高对比度或使用线稿图片")
	}
	return imageRender{Strokes: strokes, GraphicStrokes: graphicStrokes, SourceStrokes: sourceStrokes, TextRegions: regions, Warnings: warnings}, nil
}
