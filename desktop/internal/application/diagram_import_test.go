package application

import (
	"bambuscribe/desktop/internal/engine"
	"bambuscribe/desktop/internal/tableimport"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConcurrentDiagramRecognitionCompletesOrCancelsAll(t *testing.T) {
	font, err := engine.EmbeddedFont()
	if err != nil {
		t.Fatal(err)
	}
	for _, cancelBatch := range []bool{false, true} {
		t.Run(fmt.Sprint("cancel=", cancelBatch), func(t *testing.T) {
			started := make(chan struct{}, 3)
			release := make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					return
				}
				prompt, _ := json.Marshal(body["input"])
				if !strings.Contains(string(prompt), "Use the body's character spacing: 100%") {
					t.Error("recognition did not receive the body typography")
				}
				started <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				text := `{"width":100,"height":80,"strokes":[{"id":"wire","role":"wire","points":[{"x":10,"y":20},{"x":80,"y":20}],"dashed":false}],"circles":[],"labels":[],"uncertainties":[]}`
				json.NewEncoder(w).Encode(map[string]any{"status": "completed", "output": []any{map[string]any{"content": []any{map[string]any{"type": "output_text", "text": text}}}}})
			}))
			defer server.Close()
			defer unblock()
			app := &App{ctx: base, font: font, imports: importState{dir: t.TempDir()}}
			if err := app.SaveAgentConfig(tableimport.Config{BaseURL: server.URL, Model: "test", Protocol: "responses", APIKey: "test"}); err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 100, 80))); err != nil {
				t.Fatal(err)
			}
			data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
			results := make(chan error, 3)
			for range 3 {
				go func() {
					scene, err := app.RecognizeDiagram(data, "")
					if err == nil && len(scene.Strokes) != 1 {
						err = fmt.Errorf("missing diagram")
					}
					results <- err
				}()
			}
			for range 3 {
				select {
				case <-started:
				case <-time.After(3 * time.Second):
					t.Fatal("requests did not start concurrently")
				}
			}
			if _, err := app.RecognizeDiagram(data, ""); err == nil {
				t.Fatal("accepted a fourth in-flight image")
			}
			if cancelBatch {
				app.CancelDiagramRecognition()
			} else {
				unblock()
			}
			for range 3 {
				select {
				case err := <-results:
					if cancelBatch && !errors.Is(err, context.Canceled) {
						t.Fatal("request survived batch cancel", err)
					}
					if !cancelBatch && err != nil {
						t.Fatal(err)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("recognition did not finish")
				}
			}
			app.diagrams.mu.Lock()
			remaining := len(app.diagrams.recognitions)
			app.diagrams.mu.Unlock()
			if remaining != 0 {
				t.Fatal("recognition slots leaked", remaining)
			}
			// A cancelled batch must not poison the next batch or retain its slots.
			unblock()
			if _, err := app.RecognizeDiagram(data, ""); err != nil {
				t.Fatal("recognition slot was not reusable", err)
			}
		})
	}
}
