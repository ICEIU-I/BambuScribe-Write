package calculation

import (
	"fmt"
	"strings"
)

type constantDefinition struct {
	unit, source string
	values       map[string]string
}

// These are fallback values, not replacements for the reference's chosen
// precision. Air properties use the usual dry-air approximation near room
// temperature; the baseline sound speed and temperature refer specifically to 0 C.
// Sources: OpenStax University Physics 17.2 (air), CODATA 2022 (R).
var physicalConstants = map[string]constantDefinition{
	"molar_gas_constant": {
		"J/(mol·K)", "常量：摩尔气体常量，取常用近似值", map[string]string{"J/(molK)": "8.314", "J/mol/K": "8.314", "Jmol^-1K^-1": "8.314"},
	},
	"dry_air_molar_mass": {
		"kg/mol", "常量：干空气摩尔质量，取常用近似值", map[string]string{"kg/mol": "0.02897", "g/mol": "28.97", "kgmol^-1": "0.02897", "gmol^-1": "28.97"},
	},
	"dry_air_sound_speed_0c": {
		"m/s", "基准值：0°C 干空气声速，取 331.3 m/s", map[string]string{"m/s": "331.3", "ms^-1": "331.3"},
	},
	"zero_celsius_kelvin": {
		"K", "基准值：0°C 对应的绝对温度", map[string]string{"K": "273.15"},
	},
	"dry_air_heat_capacity_ratio": {
		"1", "理论值：常温干空气比热容比", map[string]string{"1": "1.4"},
	},
}

func constantUnit(unit string) string {
	return strings.NewReplacer(" ", "", "·", "", "⋅", "", "×", "", "*", "", "{", "", "}", "", "（", "(", "）", ")", "⁻¹", "^-1").Replace(unit)
}

// ResolveConstants runs once after recognition. It only supplies a missing
// value explicitly identified as a standard fallback. Unreadable reference or
// user-provided constants remain unresolved; measured quantities and
// supplied values are never guessed or overwritten. Do not call it on edits:
// a user must be able to clear a value and have normal validation catch it.
func (p Plan) ResolveConstants() (Plan, error) {
	p.Variables = append([]Variable(nil), p.Variables...)
	for i := range p.Variables {
		v := &p.Variables[i]
		if v.Constant == "" || v.Value != "" || v.Origin != "standard" {
			continue
		}
		definition, ok := physicalConstants[v.Constant]
		if !ok {
			return p, fmt.Errorf("未知常量：%s", v.Label)
		}
		unit := v.Unit
		if unit == "" {
			unit = definition.unit
		}
		value, ok := definition.values[constantUnit(unit)]
		if !ok {
			return p, fmt.Errorf("%s的常量单位无法自动换算，请使用%s或填写数值", v.Label, definition.unit)
		}
		v.Value, v.Unit, v.Source, v.Uncertain = value, unit, definition.source, false
	}
	return p, nil
}
