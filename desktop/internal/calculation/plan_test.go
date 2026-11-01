package calculation

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"
)

func TestHallCalculationsAndEditing(t *testing.T) {
	raw, err := os.ReadFile("testdata/hall.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	result, err := p.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Values) != 51 || strings.Count(result.Text, `\(`) != 51 {
		t.Fatal("missing calculations", len(result.Values))
	}
	checks := map[string]string{"u0_1": "38.1", "u0_5": "88.7", "u2_0": "35.1", "b0": "0.0290", "b7": "0.104"}
	for id, want := range checks {
		if result.Values[id] != want {
			t.Errorf("%s=%s, want %s", id, result.Values[id], want)
		}
	}
	if !strings.Contains(result.Text, "=29.0mT") || strings.Contains(result.Text, "174") {
		t.Fatal("wrong conversion or reference constant")
	}
	for i := range p.Variables {
		if p.Variables[i].ID == "kh" {
			p.Variables[i].Value = "121"
		}
	}
	changed, err := p.Calculate()
	if err != nil {
		t.Fatal(err)
	}
	if changed.Values["b0"] != "0.0580" {
		t.Fatal("dependent field not recalculated", changed.Values["b0"])
	}
	if result.Values["u2_0"] != changed.Values["u2_0"] {
		t.Fatal("editing sensitivity changed voltage")
	}
	p.Variables[0].Uncertain = true
	if _, err = p.Calculate(); err == nil {
		t.Fatal("unconfirmed reading accepted")
	}
}
func TestBoundedArithmeticAndRounding(t *testing.T) {
	for expression, want := range map[string]string{"0.1+0.2": "3/10", "(abs(-25.5)+25.3+25.4+25.4)/4": "127/5", "2^3^2": "512", "-2^2": "-4", "round(38.05,1)": "381/10", "sqrt(4)": "2"} {
		e, err := parse(expression)
		if err != nil {
			t.Fatal(err)
		}
		v, err := e.value(nil)
		if err != nil || v.RatString() != want {
			t.Errorf("%s: %v, %v", expression, v, err)
		}
	}
	for _, s := range []string{"1/0", "sqrt(-1)", "2^100", "missing+1", "0^-1", "round(1,20)", "fetch(1)", "1e999", "1e-999", "0e999999999", "0x1p2", "1;alert(1)"} {
		e, err := parse(s)
		if err == nil {
			_, err = e.value(nil)
		}
		if err == nil {
			t.Errorf("unsafe/invalid accepted: %s", s)
		}
	}
	for _, c := range []struct {
		value, format string
		precision     int
		want          string
	}{{"0", "significant", 3, "0.00"}, {"38.05", "decimal", 1, "38.1"}, {"-1.25", "decimal", 1, "-1.3"}, {"0.029008", "significant", 3, "0.0290"}, {"99.95", "significant", 3, "100"}, {"999.5", "significant", 3, "1.00e3"}} {
		v, _ := new(big.Rat).SetString(c.value)
		if got := formatValue(v, c.format, c.precision); got != c.want {
			t.Errorf("%s -> %s, want %s", c.value, got, c.want)
		}
	}
}
func TestInvalidPlanAndDependencies(t *testing.T) {
	raw, _ := os.ReadFile("testdata/hall.json")
	var p Plan
	json.Unmarshal(raw, &p)
	p.Sections[0].Steps[0].Expression = "b0+1"
	if _, err := p.Calculate(); err == nil {
		t.Fatal("forward dependency accepted")
	}
	p.Sections[0].Steps[0].ID = p.Variables[0].ID
	if err := p.Validate(); err == nil {
		t.Fatal("duplicate id accepted")
	}
}
