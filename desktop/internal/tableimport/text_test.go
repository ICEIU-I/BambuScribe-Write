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

func TestRecognizedTextUsesCompactChineseAndFormulaSpacing(t *testing.T) {
	for input, want := range map[string]string{
		"I_{S} = 10.00 mA；I_{M} = 1 000 mA；霍尔元件置于 C 型电磁铁气隙中心所在水平线上，Y_{C} = ______ mm": "I_{S}=10.00 mA；I_{M}=1000 mA；霍尔元件置于C型电磁铁气隙中心所在水平线上，Y_{C}=______ mm",
		"表 3 U_{H} 沿 X 轴 的 分布 情况":                                      "表3 U_{H}沿X轴的分布情况",
		"U̅_{H}\u00a0=\u202f-10.00\u2009mV":                            "U̅_{H}=-10.00 mV",
		"置于\u3000C\u3000型， X ＝ 1 000 mm":                               "置于C型，X＝1000 mm",
		"  Hall   effect  \n I_{S} = 10.00 mA \n\nY_{C} = ______ mm  ": "Hall effect\nI_{S}=10.00 mA\n\nY_{C}=______ mm",
		"U_{AK}/V\tI = 10⁻¹¹ A\r\n +I_{M}, -I_{S}":                     "U_{AK}/V\tI=10⁻¹¹ A\r\n+I_{M}, -I_{S}",
		"1 2 3\n1000\t2000\n10.00 mA":                                  "1 2 3\n1000\t2000\n10.00 mA",
		"":                                                             "",
	} {
		if got := normalizeText(input); got != want {
			t.Errorf("%q became %q, want %q", input, got, want)
		}
		if got := normalizeText(want); got != want {
			t.Errorf("normalizing twice changed text: %q -> %q", want, got)
		}
	}
}

func TestBothModelProtocolsNormalizeBeforeReview(t *testing.T) {
	for _, protocol := range []string{"responses", "chat-completions"} {
		t.Run(protocol, func(t *testing.T) {
			e := fixture()
			e.Tables[0].Title = "表 3 U_{H} 沿 X 轴"
			e.Tables[0].Notes = []string{"置于 C 型电磁铁， I_{M} = 1 000 mA"}
			e.Tables[0].Cells[0].Text = "I_{S} = 10.00 mA"
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				raw, _ := json.Marshal(e)
				quoted, _ := json.Marshal(string(raw))
				if protocol == "responses" {
					fmt.Fprintf(w, `{"status":"completed","output":[{"content":[{"type":"output_text","text":%s}]}]}`, quoted)
				} else {
					fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%s}}]}`, quoted)
				}
			}))
			defer server.Close()
			got, err := Recognize(context.Background(), Config{BaseURL: server.URL, Model: "test", Protocol: protocol, APIKey: "test-key"}, picture(t), "")
			if err != nil || calls != 1 {
				t.Fatal("recognition failed or requested another model call", err, calls)
			}
			want := fixture()
			want.Tables[0].Title = "表3 U_{H}沿X轴"
			want.Tables[0].Notes = []string{"置于C型电磁铁，I_{M}=1000 mA"}
			want.Tables[0].Cells[0].Text = "I_{S}=10.00 mA"
			if !reflect.DeepEqual(got, want) {
				t.Fatal("review retained model whitespace or changed topology/uncertainty", got)
			}
			// CLI/live imports can also bypass recognition and pass raw structure.
			before, _ := json.Marshal(e)
			plan, err := Build(font(t), source(), e, Placement{})
			if err != nil {
				t.Fatal(err)
			}
			if plan.Groups[0].Title != want.Tables[0].Title {
				t.Fatal("group label differs from normalized title", plan.Groups[0].Title)
			}
			for i, text := range []string{want.Tables[0].Title, want.Tables[0].Notes[0], want.Tables[0].Cells[0].Text} {
				if plan.Objects[i].Cells[0].Text != text {
					t.Fatal("direct import did not normalize text", plan.Objects[i])
				}
			}
			after, _ := json.Marshal(e)
			if string(before) != string(after) {
				t.Fatal("planning mutated the input extraction")
			}
		})
	}
}
