package calculation

import (
	"reflect"
	"strings"
	"testing"
)

func soundPlan() Plan {
	return Plan{
		Variables: []Variable{
			{ID: "M", Label: "空气摩尔质量 M", Unit: "kg/mol", Origin: "standard", Constant: "dry_air_molar_mass", Uncertain: true},
			{ID: "R", Label: "摩尔气体常量 R", Unit: "J/(mol×K)", Origin: "standard", Constant: "molar_gas_constant", Uncertain: true},
			{ID: "T", Label: "本次实验绝对温度 T", Unit: "K", Value: "289.15", Source: "本次实测16°C"},
			{ID: "v0", Label: "基准温度下的声速", Unit: "m/s", Origin: "standard", Constant: "dry_air_sound_speed_0c", Uncertain: true},
			{ID: "T0", Label: "基准绝对温度", Unit: "K", Origin: "standard", Constant: "zero_celsius_kelvin", Uncertain: true},
			{ID: "gamma_ref", Label: "比热容比理论值", Unit: "1", Origin: "standard", Constant: "dry_air_heat_capacity_ratio", Uncertain: true},
			{ID: "v", Label: "实测声速", Unit: "m/s", Value: "340"},
		},
		Sections: []Section{{Title: "空气声速与绝热系数", Steps: []Step{
			{ID: "v_theory", Label: "v", Expression: "v0*sqrt(T/T0)", Unit: "m/s", Format: "decimal", Precision: 2},
			{ID: "gamma", Label: "γ", Expression: "M*v^2/(R*T)", Format: "decimal", Precision: 4},
			{ID: "error", Label: "E", Expression: "abs(gamma-gamma_ref)/gamma_ref*100", Unit: "%", Format: "decimal", Precision: 2},
		}}},
	}
}

func TestMissingConstantsDoNotBlockSoundCalculation(t *testing.T) {
	p := soundPlan()
	if _, err := p.Calculate(); err == nil {
		t.Fatal("fixture must reproduce missing constant blocker")
	}
	resolved, err := p.ResolveConstants()
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"0.02897", "8.314", "289.15", "331.3", "273.15", "1.4", "340"} {
		v := resolved.Variables[i]
		if v.Value != want || v.Uncertain {
			t.Fatal("missing or unconfirmed value", v)
		}
		if v.Constant != "" && v.Source == "" {
			t.Fatal("missing constant provenance", v)
		}
	}
	result, err := resolved.Calculate()
	if err != nil || result.Values["v_theory"] != "340.87" || result.Values["gamma"] != "1.3931" {
		t.Fatal(result.Values, err)
	}
	if p.Variables[0].Value != "" {
		t.Fatal("mutated original plan")
	}
	resolved.Variables[0].Value = ""
	if _, err := resolved.Calculate(); err == nil {
		t.Fatal("user-cleared constant was silently reinserted")
	}
}

func TestReferenceValuesAndMissingMeasurementsArePreserved(t *testing.T) {
	p := soundPlan()
	p.Variables[0].Value = "0.029"
	p.Variables[0].Uncertain = false
	p.Variables[0].Source = "参考样例指定常量"
	p.Variables[0].Origin = "reference"
	p.Variables[3].Value = "331.45"
	p.Variables[3].Origin = "user"
	p.Variables[3].Uncertain = false
	p.Variables[2].Value = ""
	p.Variables[2].Uncertain = true
	p.Variables = append(p.Variables, Variable{ID: "sensitivity", Label: "仪器灵敏度", Unit: "mV/T"})
	resolved, err := p.ResolveConstants()
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{0, 2, 3, 7} {
		if !reflect.DeepEqual(p.Variables[i], resolved.Variables[i]) {
			t.Fatal("changed supplied value or invented measurement", i)
		}
	}
	if _, err := resolved.Calculate(); err == nil || !strings.Contains(err.Error(), "本次实验绝对温度") {
		t.Fatal("missing measured temperature must still block", err)
	}
}

func TestConstantUnitsAndUnknownIdentifiers(t *testing.T) {
	for _, tc := range []struct{ key, unit, value string }{
		{"dry_air_molar_mass", "g/mol", "28.97"},
		{"molar_gas_constant", "J/(mol·K)", "8.314"},
		{"molar_gas_constant", "J mol⁻¹ K⁻¹", "8.314"},
		{"zero_celsius_kelvin", "", "273.15"},
		{"dry_air_heat_capacity_ratio", "", "1.4"},
	} {
		p := Plan{Variables: []Variable{{Origin: "standard", Constant: tc.key, Unit: tc.unit}}}
		got, err := p.ResolveConstants()
		if err != nil || got.Variables[0].Value != tc.value || got.Variables[0].Unit == "" {
			t.Fatal(tc, got, err)
		}
	}
	for _, tc := range []Variable{
		{Origin: "standard", Constant: "unknown", Unit: "K"},
		{Origin: "standard", Constant: "zero_celsius_kelvin", Unit: "°C"},
		{Origin: "standard", Constant: "dry_air_molar_mass", Unit: "kg"},
	} {
		if _, err := (Plan{Variables: []Variable{tc}}).ResolveConstants(); err == nil {
			t.Fatal("unrecognized constant or incompatible unit accepted", tc)
		}
	}
	p := soundPlan()
	p.Variables[0].Constant = "unknown"
	if err := p.Validate(); err == nil {
		t.Fatal("unknown constant accepted in saved plan")
	}
}

func TestUnreadableReferenceConstantNeverFallsBackToStandard(t *testing.T) {
	for _, origin := range []string{"reference", "user", ""} {
		p := soundPlan()
		p.Variables[0].Origin = origin
		p.Variables[0].Source = "参考第1张常量字迹不清"
		resolved, err := p.ResolveConstants()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(p.Variables[0], resolved.Variables[0]) {
			t.Fatal("replaced an unresolved reference with default", resolved.Variables[0])
		}
		if _, err := resolved.Calculate(); err == nil {
			t.Fatal("unreadable reference bypassed confirmation")
		}
	}
}
