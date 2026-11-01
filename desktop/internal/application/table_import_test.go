package application

import (
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
	"testing"
	"time"
)

func TestConcurrentTableRecognitionCompletesOrCancelsAll(t *testing.T) {
	for _, cancelBatch := range []bool{false, true} {
		t.Run(fmt.Sprint("cancel=", cancelBatch), func(t *testing.T) {
			started := make(chan struct{}, 3)
			release := make(chan struct{})
			base, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					return
				}
				started <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				text := `{"tables":[{"title":"","notes":[],"rows":1,"columns":[1],"style":"grid","cells":[{"row":0,"col":0,"rowSpan":1,"colSpan":1,"text":"A","uncertain":false,"borders":null}]}]}`
				q, _ := json.Marshal(text)
				fmt.Fprintf(w, `{"status":"completed","output":[{"content":[{"type":"output_text","text":%s}]}]}`, q)
			}))
			defer server.Close()
			defer close(release)
			app := &App{ctx: base, imports: importState{dir: t.TempDir()}}
			if err := app.SaveAgentConfig(tableimport.Config{BaseURL: server.URL, Model: "test", Protocol: "responses", APIKey: "test"}); err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10, 10)))
			data := "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
			results := make(chan error, 3)
			for range 3 {
				go func() {
					e, err := app.RecognizeTables(data, "")
					if err == nil && len(e.Tables) != 1 {
						err = fmt.Errorf("missing table")
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
			if _, err := app.RecognizeTables(data, ""); err == nil {
				t.Fatal("accepted a fourth in-flight image")
			}
			if cancelBatch {
				app.CancelTableRecognition()
			} else {
				for range 3 {
					release <- struct{}{}
				}
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
			app.imports.mu.Lock()
			remaining := len(app.imports.recognitions)
			app.imports.mu.Unlock()
			if remaining != 0 {
				t.Fatal("recognition slots leaked", remaining)
			}
		})
	}
}
