package localocr

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDisabledAndInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BAMBUSCRIBE_CONFIG_DIR", dir)
	if _, enabled, err := Recognize(nil); enabled || err != nil {
		t.Fatalf("missing config: %v %v", enabled, err)
	}
	os.WriteFile(filepath.Join(dir, "image-ocr.json"), []byte(`{"enabled":false}`), 0600)
	if _, enabled, err := Recognize(nil); enabled || err != nil {
		t.Fatal(enabled, err)
	}
	os.WriteFile(filepath.Join(dir, "image-ocr.json"), []byte(`{"enabled":true,"python":"relative"}`), 0600)
	if _, enabled, err := Recognize(nil); !enabled || err == nil {
		t.Fatal("invalid configuration accepted")
	}
	os.WriteFile(filepath.Join(dir, "image-ocr.json"), []byte(`{`), 0600)
	if _, enabled, err := Recognize(nil); !enabled || err == nil {
		t.Fatal("malformed config silently disabled")
	}
}

func TestResultValidation(t *testing.T) {
	good := `[{"text":"F_{E}","x":0.1,"y":0.2,"w":0.1,"h":0.1,"confidence":0.9,"replace":true,"textFormat":"math","backend":"local-hybrid"}]`
	for _, raw := range []string{good, `[]`} {
		if _, err := decode([]byte(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for _, raw := range []string{good + `[]`, strings.Replace(good, `"w":0.1`, `"w":2`, 1), strings.Replace(good, `"confidence":0.9`, `"confidence":-1`, 1), strings.Replace(good, `"math"`, `"latex"`, 1), strings.Replace(good, `"local-hybrid"`, `"other"`, 1), "[" + strings.Repeat(good[1:len(good)-1]+",", 160) + good[1:len(good)-1] + "]"} {
		if _, err := decode([]byte(raw)); err == nil {
			t.Fatal("invalid result accepted", raw[:min(len(raw), 150)])
		}
	}
}

func fakeConfig(t *testing.T, script string) (Config, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX worker fixture")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "python")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	c := Config{Enabled: true, Python: exe, FormulaPython: exe, ModelDir: dir, FormulaModelDir: dir}
	for _, name := range []string{"PP-OCRv6_det_medium.onnx", "PP-OCRv6_rec_medium.onnx", "encoder_model.onnx", "decoder_model.onnx"} {
		os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600)
	}
	raw, _ := json.Marshal(c)
	os.WriteFile(filepath.Join(dir, "image-ocr.json"), raw, 0600)
	t.Setenv("BAMBUSCRIBE_CONFIG_DIR", dir)
	return c, dir
}

func TestCacheReuseSourceAndModelInvalidation(t *testing.T) {
	_, dir := fakeConfig(t, `printf '[]' > "$3"`)
	if _, _, err := Recognize([]byte("one")); err != nil {
		t.Fatal(err)
	}
	// A cached source must not launch a broken worker.
	os.WriteFile(filepath.Join(dir, "python"), []byte("#!/bin/sh\nexit 1\n"), 0700)
	if _, _, err := Recognize([]byte("one")); err != nil {
		t.Fatal("cache not reused", err)
	}
	if _, _, err := Recognize([]byte("two")); err == nil {
		t.Fatal("different image reused cache")
	}
	os.WriteFile(filepath.Join(dir, "PP-OCRv6_rec_medium.onnx"), []byte("new weights"), 0600)
	if _, _, err := Recognize([]byte("one")); err == nil {
		t.Fatal("changed model reused cache")
	}
	files, _ := os.ReadDir(filepath.Join(dir, "image-ocr-cache"))
	if len(files) != 1 {
		t.Fatal("failed results cached", len(files))
	}
}

func TestWorkerFailureAndCancellation(t *testing.T) {
	c, _ := fakeConfig(t, `printf 'invalid' > "$3"`)
	if _, err := run(context.Background(), c, []byte("image")); err == nil {
		t.Fatal("invalid output accepted")
	}
	os.WriteFile(c.Python, []byte("#!/bin/sh\nexec sleep 5\n"), 0700)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := run(ctx, c, nil); err == nil || time.Since(start) > time.Second {
		t.Fatal("worker did not stop at deadline", err)
	}
}

func TestCacheBound(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 40; i++ {
		name := strings.Repeat("a", 62) + string("0123456789"[i/10]) + string("0123456789"[i%10]) + ".json"
		os.WriteFile(filepath.Join(dir, name), []byte("[]"), 0600)
	}
	prune(dir)
	files, _ := os.ReadDir(dir)
	if len(files) != 32 {
		t.Fatal("cache not bounded", len(files))
	}
}
