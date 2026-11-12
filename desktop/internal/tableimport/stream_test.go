package tableimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func responseSnapshot(text string) string {
	quoted, _ := json.Marshal(text)
	return fmt.Sprintf(`{"status":"completed","output":[{"content":[{"type":"output_text","text":%s}]}]}`, quoted)
}

func TestStructuredStreamsPreserveTextAndReportProgress(t *testing.T) {
	for _, protocol := range []string{"responses", "chat-completions"} {
		t.Run(protocol, func(t *testing.T) {
			text := `{"formula":"声速=λ×f","value":340}`
			stages := []string{}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["stream"] != true || !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
					t.Error("streaming was not requested")
				}
				w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
				fmt.Fprint(w, ": heartbeat\r\n\r\n")
				w.(http.Flusher).Flush()
				for _, part := range []string{text[:8], text[8:]} {
					quoted, _ := json.Marshal(part)
					if protocol == "responses" {
						// Event names and multi-line data are valid SSE framing.
						fmt.Fprintf(w, "event: response.output_text.delta\r\ndata: {\r\ndata: \"delta\":%s}\r\n\r\n", quoted)
					} else {
						fmt.Fprintf(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":%s},\"finish_reason\":null}]}\n\n", quoted)
					}
					w.(http.Flusher).Flush()
				}
				if protocol == "responses" {
					fmt.Fprintf(w, "data: {\"type\":\"response.completed\",\"response\":%s}\n\n", responseSnapshot(text))
				} else {
					fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
				}
			}))
			defer server.Close()
			ctx := WithProgress(context.Background(), func(stage string) { stages = append(stages, stage) })
			got, err := RequestStructured(ctx, Config{BaseURL: server.URL, Protocol: protocol, Model: "test", APIKey: "test-secret"}, nil, "", "", json.RawMessage(`{}`), "test", 100)
			if err != nil || got != text || calls != 1 {
				t.Fatalf("got %q, err %v, calls %d", got, err, calls)
			}
			if !reflect.DeepEqual(stages, []string{"processing", "receiving"}) {
				t.Fatal("missing or duplicated progress", stages)
			}
		})
	}
}

func TestStreamRejectsPartialFailedAndOversizedResults(t *testing.T) {
	for _, tc := range []struct{ name, protocol, body string }{
		{"response EOF", "responses", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"{}\"}\n\n"},
		{"response done without completion", "responses", "data: [DONE]\n\n"},
		{"response failed", "responses", "data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"test-secret\"}}}\n\n"},
		{"response incomplete", "responses", "data: {\"type\":\"response.incomplete\"}\n\n"},
		{"refusal", "responses", "data: {\"type\":\"response.refusal.delta\",\"delta\":\"test-secret\"}\n\n"},
		{"chat EOF", "chat-completions", "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"{}\"}}]}\n\n"},
		{"chat length", "chat-completions", "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"{}\"},\"finish_reason\":\"length\"}]}\n\n"},
		{"chat other choice", "chat-completions", "data: {\"choices\":[{\"index\":1,\"delta\":{\"content\":\"{}\"},\"finish_reason\":\"stop\"}]}\n\n"},
		{"gateway error", "chat-completions", "data: {\"error\":{\"message\":\"test-secret\"}}\n\n"},
		{"invalid event", "responses", "data: test-secret\n\n"},
		{"oversized event", "responses", "data: " + strings.Repeat("x", maxResponseBytes) + "\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readModelResponse(context.Background(), strings.NewReader(tc.body), "text/event-stream", tc.protocol)
			if err == nil || got != "" || strings.Contains(err.Error(), "test-secret") {
				t.Fatalf("accepted incomplete output or leaked details: %q %v", got, err)
			}
		})
	}
}

func TestCompatibleJSONAndMissingStreamContentType(t *testing.T) {
	snapshot := responseSnapshot(`{"ok":true}`)
	for _, body := range []string{snapshot, "event: response.completed\ndata: {\"response\":" + snapshot + "}\n\n"} {
		got, err := readModelResponse(context.Background(), strings.NewReader(body), "", "responses")
		if err != nil || got != `{"ok":true}` {
			t.Fatal(got, err)
		}
	}
}

func TestCancelAndTimeoutAfterResponseHeaders(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint("deadline=", deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 150*time.Millisecond)
			}
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, ": heartbeat\n\n")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer server.Close()
			progress := WithProgress(ctx, func(string) {
				if !deadline {
					cancel()
				}
			})
			got, err := RequestStructured(progress, Config{BaseURL: server.URL, Protocol: "responses"}, nil, "", "", json.RawMessage(`{}`), "test", 100)
			want := context.Canceled
			if deadline {
				want = context.DeadlineExceeded
				if !strings.Contains(err.Error(), "超时") {
					t.Fatal("timeout lost user-facing explanation", err)
				}
			}
			if got != "" || !errors.Is(err, want) {
				t.Fatalf("got %q %v, want %v", got, err, want)
			}
		})
	}
}

func TestGatewayTimeoutDoesNotBlameModelOrRetry(t *testing.T) {
	for _, status := range []int{408, 504, 524} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(status)
			fmt.Fprint(w, "test-secret")
		}))
		_, err := RequestStructured(context.Background(), Config{BaseURL: server.URL, Protocol: "responses"}, nil, "", "", json.RawMessage(`{}`), "test", 100)
		server.Close()
		if err == nil || !strings.Contains(err.Error(), "超时") || strings.Contains(err.Error(), "协议") || strings.Contains(err.Error(), "test-secret") || calls != 1 {
			t.Fatal(status, calls, err)
		}
	}
}
