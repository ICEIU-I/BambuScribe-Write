package calculationimport

import (
	"bambuscribe/desktop/internal/tableimport"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestBothProtocolsKeepImageRolesInSingleRequest(t *testing.T) {
	raw, err := os.ReadFile("../engine/testdata/ocr-formula-table.png")
	if err != nil {
		t.Fatal(err)
	}
	image := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
	plan, err := os.ReadFile("../calculation/testdata/hall.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, protocol := range []string{"responses", "chat-completions"} {
		t.Run(protocol, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("Authorization") != "Bearer local-test" {
					t.Error("missing authorization")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(body)
				text := string(encoded)
				for _, label := range []string{"原始数据第1张", "原始数据第2张", "参考样例第1张", "参考样例第2张", "灵敏度242"} {
					if !strings.Contains(text, label) {
						t.Errorf("missing source role: %s", label)
					}
				}
				if strings.Index(text, "原始数据第1张") > strings.Index(text, "原始数据第2张") || strings.Index(text, "参考样例第1张") > strings.Index(text, "参考样例第2张") {
					t.Error("image order changed")
				}
				if protocol == "responses" {
					json.NewEncoder(w).Encode(map[string]any{"status": "completed", "output": []any{map[string]any{"content": []any{map[string]any{"type": "output_text", "text": string(plan)}}}}})
				} else {
					json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": string(plan)}}}})
				}
			}))
			defer server.Close()
			p, err := Recognize(context.Background(), tableimport.Config{BaseURL: server.URL, Model: "test", Protocol: protocol, APIKey: "local-test"}, Sources{Data: []string{image, image}, References: []string{image, image}, Instruction: "灵敏度242"})
			if err != nil || calls != 1 || len(p.Sections) != 4 {
				t.Fatal("recognition failed", err, calls)
			}
		})
	}
}
func TestInvalidSourcesMakeNoRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	c := tableimport.Config{BaseURL: server.URL, Model: "test", Protocol: "responses", APIKey: "local-test"}
	if _, err := Recognize(context.Background(), c, Sources{Data: []string{"invalid"}, References: []string{"invalid"}}); err == nil {
		t.Fatal("invalid image accepted")
	}
	if calls != 0 {
		t.Fatal("invalid data sent")
	}
}
