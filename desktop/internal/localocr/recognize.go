package localocr

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

//go:embed worker/main.py worker/formula.py worker/notation.py worker/regions.py worker/fraction_policy.py
var workers embed.FS
var recognitionMu sync.Mutex

// Recognize is bounded and cached by source bytes, worker code and local config.
// An unavailable worker is reported to the caller for native fallback; failed
// attempts are not cached as successful recognition.
func Recognize(data []byte) ([]Box, bool, error) {
	c, dir, err := configuration()
	if !c.Enabled {
		return nil, false, err
	}
	if err != nil {
		return nil, true, err
	}
	if err = c.validate(); err != nil {
		return nil, true, err
	}
	recognitionMu.Lock()
	defer recognitionMu.Unlock()
	h := sha256.New()
	h.Write(data)
	raw, _ := json.Marshal(c)
	h.Write(raw)
	// Replacing a model at the same path must invalidate its previous results.
	for _, path := range []string{filepath.Join(c.ModelDir, "PP-OCRv6_det_medium.onnx"), filepath.Join(c.ModelDir, "PP-OCRv6_rec_medium.onnx"), filepath.Join(c.FormulaModelDir, "encoder_model.onnx"), filepath.Join(c.FormulaModelDir, "decoder_model.onnx")} {
		info, _ := os.Stat(path)
		if info != nil {
			fmt.Fprintf(h, "%s:%d:%d", path, info.Size(), info.ModTime().UnixNano())
		}
	}
	files, _ := workers.ReadDir("worker")
	for _, file := range files {
		raw, _ := workers.ReadFile("worker/" + file.Name())
		h.Write(raw)
	}
	cacheDir := filepath.Join(dir, "image-ocr-cache")
	cacheFile := filepath.Join(cacheDir, hex.EncodeToString(h.Sum(nil))+".json")
	if info, err := os.Stat(cacheFile); err == nil && info.Size() <= 1<<20 {
		raw, _ := os.ReadFile(cacheFile)
		if boxes, err := decode(raw); err == nil {
			return boxes, true, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	boxes, err := run(ctx, c, data)
	if err != nil {
		return nil, true, err
	}
	if os.MkdirAll(cacheDir, 0700) == nil {
		raw, _ := json.Marshal(boxes)
		if f, e := os.CreateTemp(cacheDir, ".ocr-"); e == nil {
			name := f.Name()
			_, e = f.Write(raw)
			closeErr := f.Close()
			if e == nil && closeErr == nil {
				_ = os.Rename(name, cacheFile)
			}
			_ = os.Remove(name)
		}
		prune(cacheDir)
	}
	return boxes, true, nil
}

func run(ctx context.Context, c Config, data []byte) ([]Box, error) {
	dir, err := os.MkdirTemp("", "bambuscribe-ocr-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	files, _ := workers.ReadDir("worker")
	for _, file := range files {
		raw, _ := workers.ReadFile("worker/" + file.Name())
		if err = os.WriteFile(filepath.Join(dir, file.Name()), raw, 0600); err != nil {
			return nil, err
		}
	}
	imagePath := filepath.Join(dir, "source.png")
	if err = os.WriteFile(imagePath, data, 0600); err != nil {
		return nil, err
	}
	request := struct {
		Config
		Image string `json:"image"`
	}{c, imagePath}
	raw, _ := json.Marshal(request)
	input, output := filepath.Join(dir, "request.json"), filepath.Join(dir, "result.json")
	if err = os.WriteFile(input, raw, 0600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, c.Python, filepath.Join(dir, "main.py"), input, output)
	cmd.Env = append(os.Environ(), "HF_HUB_OFFLINE=1", "HF_HUB_DISABLE_TELEMETRY=1", "OMP_NUM_THREADS=4")
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err = cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("本机 OCR 识别超时")
		}
		return nil, fmt.Errorf("本机 OCR 进程未完成识别")
	}
	info, err := os.Stat(output)
	if err != nil || info.Size() > 1<<20 {
		return nil, fmt.Errorf("本机 OCR 返回结果无效")
	}
	raw, err = os.ReadFile(output)
	if err != nil {
		return nil, err
	}
	return decode(raw)
}

func decode(raw []byte) ([]Box, error) {
	var boxes []Box
	if err := json.Unmarshal(raw, &boxes); err != nil {
		return nil, err
	}
	if len(boxes) > 160 {
		return nil, fmt.Errorf("OCR 区域过多")
	}
	for _, b := range boxes {
		finite := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
		if len([]rune(b.Text)) > 500 || !finite(b.X) || !finite(b.Y) || !finite(b.W) || !finite(b.H) || b.X < 0 || b.Y < 0 || b.W <= 0 || b.H <= 0 || b.X+b.W > 1.000001 || b.Y+b.H > 1.000001 || !finite(b.Confidence) || b.Confidence < 0 || b.Confidence > 1 || b.Backend != "local-hybrid" || b.TextFormat != "" && b.TextFormat != "math" {
			return nil, fmt.Errorf("OCR 区域数据无效")
		}
	}
	return boxes, nil
}

func prune(dir string) {
	entries, _ := os.ReadDir(dir)
	type cached struct {
		name string
		time time.Time
	}
	files := []cached{}
	for _, e := range entries {
		if len(e.Name()) == 69 && filepath.Ext(e.Name()) == ".json" {
			if info, err := e.Info(); err == nil {
				files = append(files, cached{e.Name(), info.ModTime()})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].time.Before(files[j].time) })
	for len(files) > 32 {
		_ = os.Remove(filepath.Join(dir, files[0].name))
		files = files[1:]
	}
}
