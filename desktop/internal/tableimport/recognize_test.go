package tableimport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRecognitionKeepsUncertainAndUnsupportedTextWithoutExtraRequests(t *testing.T) {
	want := fixture()
	want.Tables[0].Cells[0].Text = "U̅_{H} 🧪"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := json.Marshal(want)
		q, _ := json.Marshal(string(b))
		fmt.Fprintf(w, `{"status":"completed","output":[{"content":[{"type":"output_text","text":%s}]}]}`, q)
	}))
	defer server.Close()
	got, err := Recognize(context.Background(), Config{BaseURL: server.URL, Model: "test", Protocol: "responses", APIKey: "secret"}, picture(t), "")
	if err != nil || calls != 1 || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err, calls)
	}
	plan, err := Build(font(t), source(), got, Placement{})
	if err != nil || len(plan.Warnings) < 2 {
		t.Fatal("lost local uncertainty/missing-glyph checks", plan.Warnings, err)
	}
}
func TestInvalidRecognitionFailsLocallyWithoutRepairRequests(t *testing.T) {
	for _, invalidJSON := range []bool{false, true} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			e := fixture()
			e.Tables[0].Cells[0].ColSpan = 2
			b, _ := json.Marshal(e)
			if invalidJSON {
				b = []byte("not JSON")
			}
			q, _ := json.Marshal(string(b))
			fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%s}}]}`, q)
		}))
		got, err := Recognize(context.Background(), Config{BaseURL: server.URL, Model: "test", Protocol: "chat-completions", APIKey: "secret"}, picture(t), "")
		server.Close()
		if calls != 1 || err == nil || len(got.Tables) > 0 {
			t.Fatal(calls, err, got)
		}
	}
}
