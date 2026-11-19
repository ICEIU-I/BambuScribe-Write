package diagramagent

import (
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func writeSceneStream(w http.ResponseWriter, protocol, scene string, complete bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, ": heartbeat\n\n")
	w.(http.Flusher).Flush()
	for _, part := range []string{scene[:len(scene)/2], scene[len(scene)/2:]} {
		quoted, _ := json.Marshal(part)
		if protocol == "responses" {
			fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":%s}\n\n", quoted)
		} else {
			fmt.Fprintf(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":%s}}]}\n\n", quoted)
		}
		w.(http.Flusher).Flush()
	}
	if !complete {
		return
	}
	if protocol == "responses" {
		quoted, _ := json.Marshal(scene)
		fmt.Fprintf(w, "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"content\":[{\"type\":\"output_text\",\"text\":%s}]}]}}\n\n", quoted)
	} else {
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}
}

func TestDiagramRecognitionStreamsAndValidatesCompletion(t *testing.T) {
	for _, protocol := range []string{"responses", "chat-completions"} {
		for _, tc := range []struct {
			name, scene string
			complete    bool
			wantError   string
		}{
			{"complete", sample, true, ""},
			{"disconnect after valid JSON", sample, false, "连接中断"},
			{"changed dimensions", strings.Replace(sample, `"width":100`, `"width":101`, 1), true, "原图坐标"},
		} {
			t.Run(protocol+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					if body["stream"] != true || !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
						t.Error("diagram request must enable streaming")
					}
					if protocol == "responses" {
						if body["reasoning"].(map[string]any)["effort"] != "medium" || body["max_output_tokens"] != float64(18000) {
							t.Error("diagram reasoning or output budget changed")
						}
					} else if body["reasoning_effort"] != "medium" || body["max_completion_tokens"] != float64(18000) {
						t.Error("diagram reasoning or output budget changed")
					}
					writeSceneStream(w, protocol, tc.scene, tc.complete)
				}))
				defer server.Close()
				config := tableimport.Config{BaseURL: server.URL, APIKey: "fixture-key", Model: "gpt-6.1-sol", Protocol: protocol}
				scene, err := Recognize(context.Background(), config, picture(t), "保留间隙")
				if tc.wantError != "" {
					if err == nil || !strings.Contains(err.Error(), tc.wantError) || scene != nil {
						t.Fatalf("want %q, got scene=%v err=%v", tc.wantError, scene, err)
					}
				} else if err != nil || scene.Width != 100 || len(scene.Strokes) != 1 || len(scene.Strokes[0].Points) != 2 {
					t.Fatalf("scene lost or duplicated: %v, %v", scene, err)
				}
				if calls.Load() != 1 {
					t.Fatal("unexpected automatic retry", calls.Load())
				}
			})
		}
	}
}

func TestDiagramGatewayTimeoutExplainsFailureWithoutRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(524)
		fmt.Fprint(w, "secret fixture-key")
	}))
	defer server.Close()
	config := tableimport.Config{BaseURL: server.URL, APIKey: "fixture-key", Model: "test", Protocol: "responses"}
	scene, err := Recognize(context.Background(), config, picture(t), "")
	if scene != nil || err == nil || !strings.Contains(err.Error(), "524") || !strings.Contains(err.Error(), "超时") || strings.Contains(err.Error(), "额度") || strings.Contains(err.Error(), "fixture-key") || calls.Load() != 1 {
		t.Fatalf("wrong timeout explanation or retry: %v, calls=%d", err, calls.Load())
	}
}

func TestDiagramStreamCancellation(t *testing.T) {
	for _, protocol := range []string{"responses", "chat-completions"} {
		t.Run(protocol, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, ": heartbeat\n\n")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer server.Close()
			// Cancel after headers are received, while the model is still generating.
			ctx = tableimport.WithProgress(ctx, func(string) { cancel() })
			config := tableimport.Config{BaseURL: server.URL, APIKey: "fixture-key", Model: "test", Protocol: protocol}
			scene, err := Recognize(ctx, config, picture(t), "")
			if scene != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v, %v", scene, err)
			}
		})
	}
}
