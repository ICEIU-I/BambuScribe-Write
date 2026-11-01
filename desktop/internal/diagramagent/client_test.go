package diagramagent

import (
	"bambuscribe/desktop/internal/tableimport"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func picture(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 100, 80))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

const sample = `{"width":100,"height":80,"strokes":[{"id":"wire","role":"wire","points":[{"x":10,"y":20},{"x":80,"y":20}],"dashed":false}],"circles":[],"labels":[],"uncertainties":[]}`

func TestSingleRequestWithSourceAndTwoCrops(t *testing.T) {
	for _, protocol := range []string{"responses", "chat-completions"} {
		t.Run(protocol, func(t *testing.T) {
			var count atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count.Add(1)
				if r.Header.Get("Authorization") != "Bearer fixture-key" {
					t.Error("missing auth")
				}
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if body["model"] != "gpt-6.1-sol" {
					t.Error("model changed")
				}
				var content []any
				if protocol == "responses" {
					content = body["input"].([]any)[0].(map[string]any)["content"].([]any)
					json.NewEncoder(w).Encode(map[string]any{"status": "completed", "output": []any{map[string]any{"content": []any{map[string]any{"type": "output_text", "text": sample}}}}})
				} else {
					content = body["messages"].([]any)[1].(map[string]any)["content"].([]any)
					json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": sample}}}})
				}
				if len(content) != 4 {
					t.Error("expected prompt + 3 images")
				}
			}))
			defer server.Close()
			c := tableimport.Config{BaseURL: server.URL, APIKey: "fixture-key", Model: "gpt-6.1-sol", Protocol: protocol}
			result, err := Recognize(context.Background(), c, picture(t), "保留间隙")
			if err != nil || result.Width != 100 {
				t.Fatal(result, err)
			}
			if count.Load() != 1 {
				t.Fatal("unexpected review/retry request", count.Load())
			}
		})
	}
}

func TestFailureDoesNotRetryOrLeakResponse(t *testing.T) {
	var count atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(403)
		w.Write([]byte("secret bearer fixture-key"))
	}))
	defer s.Close()
	c := tableimport.Config{BaseURL: s.URL, APIKey: "fixture-key", Model: "test", Protocol: "responses"}
	_, err := Recognize(context.Background(), c, picture(t), "")
	if err == nil || strings.Contains(err.Error(), "fixture-key") || count.Load() != 1 {
		t.Fatal(err, count.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Recognize(ctx, c, picture(t), "")
	if err != context.Canceled || count.Load() != 1 {
		t.Fatal("cancelled request was sent", err)
	}
}
