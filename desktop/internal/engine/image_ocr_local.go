package engine

import "bambuscribe/desktop/internal/localocr"

// OCRBox uses normalized source-image coordinates on every platform.
type OCRBox struct {
	Text       string  `json:"text"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	W          float64 `json:"w"`
	H          float64 `json:"h"`
	Confidence float64 `json:"confidence"`
	Backend    string  `json:"backend,omitempty"`
	Replace    bool    `json:"replace,omitempty"`
	TextFormat string  `json:"textFormat,omitempty"`
}

func recognizeLocalOrNative(raw []byte, native func() []OCRBox) ([]OCRBox, string) {
	boxes, enabled, err := localocr.Recognize(raw)
	if !enabled {
		return native(), ""
	}
	if err != nil {
		return native(), "本机 OCR 暂不可用，已使用系统识别；请检查 image-ocr.json 中的运行环境和模型路径后重试。"
	}
	if len(boxes) == 0 {
		return native(), ""
	}
	out := make([]OCRBox, len(boxes))
	for i, b := range boxes {
		out[i] = OCRBox{Text: b.Text, X: b.X, Y: b.Y, W: b.W, H: b.H, Confidence: b.Confidence, Backend: b.Backend, Replace: b.Replace, TextFormat: b.TextFormat}
	}
	return out, ""
}
