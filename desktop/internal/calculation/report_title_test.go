package calculation

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestReportTitlesRemoveProcessNotesWithoutChangingScience(t *testing.T) {
	cases := map[string]string{
		"34°C：速度、黏度；按参考在Re＜0.1时取修正黏度等于原黏度": "34°C：速度、黏度",
		"平均值; 根据参考样例保留三位有效数字":              "平均值",
		"根据样例计算平均时间":                       "计算平均时间",
		"黏度（按参考）":                          "黏度",
		"按参考":                              "",
		"参考值与测量值的相对误差":                     "参考值与测量值的相对误差",
		"按参考值计算相对误差":                       "按参考值计算相对误差",
		"黏度；Re＜0.1，忽略雷诺数修正":                "黏度；Re＜0.1，忽略雷诺数修正",
	}
	for input, want := range cases {
		if got := cleanReportTitle(input); got != want {
			t.Errorf("%q => %q, want %q", input, got, want)
		}
		if cleanReportTitle(want) != want {
			t.Errorf("cleanup is not idempotent: %q", want)
		}
	}
}

func TestSavedPlanRecalculationDoesNotRestoreGenerationNotes(t *testing.T) {
	raw, _ := os.ReadFile("testdata/hall.json")
	p, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := p.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	originalTitle := p.Sections[0].Title
	p.Sections[0].Title += "；按参考保留计算步骤"
	p.Variables[0].Source = "参考样例中的常量位置"
	clean := p.CleanReportTitles()
	if clean.Sections[0].Title != originalTitle || !strings.Contains(p.Sections[0].Title, "按参考") {
		t.Fatal("clean title missing or input plan mutated")
	}
	if !reflect.DeepEqual(p.Variables, clean.Variables) || !reflect.DeepEqual(p.Sections[0].Steps, clean.Sections[0].Steps) {
		t.Fatal("provenance or calculations changed")
	}
	// Older CCF records pass their original plan straight to Calculate.
	got, err := p.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != baseline.Text || !reflect.DeepEqual(got.Values, baseline.Values) {
		t.Fatal("recalculation restored process notes or changed numeric results")
	}
}
