package tableimport

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func picture(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10, 10)))
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestImageRequestUsesSelectedModelAndStrictSchema(t *testing.T) {
	for _, protocol := range []string{"responses", "chat-completions"} {
		t.Run(protocol, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "Bearer test-secret" {
					t.Error("missing auth")
				}
				var body map[string]any
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Error("invalid body")
				}
				if body["model"] != "gpt-6.1-sol" {
					t.Error("model substituted")
				}
				raw, _ := json.Marshal(body)
				if !bytes.Contains(raw, []byte("data:image/png;base64,")) || !bytes.Contains(raw, []byte(`"strict":true`)) {
					t.Error("lost image/schema")
				}
				if calls > 1 {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				text, _ := json.Marshal(fixture())
				quoted, _ := json.Marshal(string(text))
				if protocol == "responses" {
					fmt.Fprintf(w, `{"status":"completed","output":[{"content":[{"type":"output_text","text":%s}]}]}`, quoted)
				} else {
					fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%s}}]}`, quoted)
				}
			}))
			defer server.Close()
			got, err := Recognize(context.Background(), Config{BaseURL: server.URL + "/v1", Model: "gpt-6.1-sol", Protocol: protocol, APIKey: "test-secret"}, picture(t), "留空")
			if err != nil || len(got.Tables) != 1 || calls != 1 || got.Review != nil {
				t.Fatal(got, err)
			}
		})
	}
}
func TestClientRejectsIncompleteOutputAndRedactsErrors(t *testing.T) {
	for _, status := range []int{400, 401, 429, 200} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"choices":[{"finish_reason":"length","message":{"content":"test-secret"}}]}`))
		}))
		_, err := Recognize(context.Background(), Config{BaseURL: server.URL, Model: "test", Protocol: "chat-completions", APIKey: "test-secret"}, picture(t), "")
		server.Close()
		if err == nil || strings.Contains(err.Error(), "test-secret") {
			t.Fatal("missing error or leaked secret", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Recognize(ctx, Config{BaseURL: "http://127.0.0.1:1", Model: "test", Protocol: "responses", APIKey: "key"}, picture(t), ""); err != context.Canceled {
		t.Fatal(err)
	}
}
func TestConfigKeepsKeyOutsideSnapshot(t *testing.T) {
	dir := t.TempDir()
	c := Config{BaseURL: "https://example.com/v1", Model: "test", Protocol: "responses", APIKey: "secret"}
	if err := SaveConfig(dir, c); err != nil {
		t.Fatal(err)
	}
	c.APIKey = ""
	c.Model = "updated"
	if err := SaveConfig(dir, c); err != nil {
		t.Fatal(err)
	}
	saved, err := LoadConfig(dir)
	if err != nil || saved.APIKey != "secret" {
		t.Fatal(err)
	}
	view := Redacted(saved)
	b, _ := json.Marshal(view)
	if strings.Contains(string(b), "secret") || !view.HasKey {
		t.Fatal("key returned to webview")
	}
	info, _ := os.Stat(filepath.Join(dir, "table-agent.json"))
	if info.Mode().Perm() != 0600 {
		t.Fatal("key permissions")
	}
}
