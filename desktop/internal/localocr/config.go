// Package localocr runs the optional, offline image OCR trial independently of
// application storage. Model paths and caches never enter a saved project.
package localocr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Enabled         bool   `json:"enabled"`
	Python          string `json:"python"`
	FormulaPython   string `json:"formulaPython"`
	ModelDir        string `json:"modelDir"`
	FormulaModelDir string `json:"formulaModelDir"`
}

type Box struct {
	Text       string  `json:"text"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	W          float64 `json:"w"`
	H          float64 `json:"h"`
	Confidence float64 `json:"confidence"`
	Replace    bool    `json:"replace"`
	TextFormat string  `json:"textFormat"`
	Backend    string  `json:"backend"`
}

func configuration() (Config, string, error) {
	dir := os.Getenv("BAMBUSCRIBE_CONFIG_DIR")
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return Config{}, "", err
		}
		dir = filepath.Join(base, "BambuScribeWrite")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "image-ocr.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, dir, nil
	}
	if err != nil {
		return Config{Enabled: true}, dir, err
	}
	var c Config
	if err = json.Unmarshal(raw, &c); err != nil {
		return Config{Enabled: true}, dir, err
	}
	return c, dir, nil
}

func (c Config) validate() error {
	for _, p := range []string{c.Python, c.FormulaPython, filepath.Join(c.ModelDir, "PP-OCRv6_det_medium.onnx"), filepath.Join(c.ModelDir, "PP-OCRv6_rec_medium.onnx"), filepath.Join(c.FormulaModelDir, "encoder_model.onnx"), filepath.Join(c.FormulaModelDir, "decoder_model.onnx")} {
		if !filepath.IsAbs(p) {
			return fmt.Errorf("OCR 路径须为绝对路径")
		}
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			return fmt.Errorf("本机 OCR 运行环境或模型缺失")
		}
	}
	return nil
}
