package calculation

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"strings"
)

type Variable struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	Unit      string `json:"unit"`
	Source    string `json:"source"`
	Uncertain bool   `json:"uncertain"`
}
type Conversion struct {
	Factor    string `json:"factor"`
	Unit      string `json:"unit"`
	Precision int    `json:"precision"`
	Format    string `json:"format"`
}
type Step struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Symbolic    string       `json:"symbolic"`
	Expression  string       `json:"expression"`
	Unit        string       `json:"unit"`
	Precision   int          `json:"precision"`
	Format      string       `json:"format"`
	UseRounded  bool         `json:"useRounded"`
	Conversions []Conversion `json:"conversions"`
}
type Section struct {
	Title string `json:"title"`
	Steps []Step `json:"steps"`
}
type Plan struct {
	Variables []Variable `json:"variables"`
	Sections  []Section  `json:"sections"`
	Issues    []string   `json:"issues"`
}
type Result struct {
	Text   string            `json:"text"`
	Values map[string]string `json:"values"`
}

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z_0-9]{0,39}$`)

func Decode(raw []byte) (Plan, error) {
	var p Plan
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return p, fmt.Errorf("识别结果格式无效，请重试")
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return p, fmt.Errorf("识别结果包含多余内容，请重试")
	}
	return p, p.Validate()
}
func (p Plan) Validate() error {
	if len(p.Variables) > 600 || len(p.Sections) < 1 || len(p.Sections) > 30 || len(p.Issues) > 30 {
		return fmt.Errorf("计算内容过多或为空，请分批导入")
	}
	ids := map[string]bool{}
	count, chars := 0, 0
	checkID := func(id string) error {
		if !identifier.MatchString(id) || ids[id] {
			return fmt.Errorf("变量或计算项编号重复/无效：%s", id)
		}
		ids[id] = true
		return nil
	}
	for _, v := range p.Variables {
		if err := checkID(v.ID); err != nil {
			return err
		}
		chars += len(v.Label) + len(v.Source) + len(v.Unit) + len(v.Value)
		if v.Value != "" {
			if _, err := number(v.Value); err != nil {
				return fmt.Errorf("%s：%w", v.Label, err)
			}
		}
	}
	for _, s := range p.Sections {
		chars += len(s.Title)
		if strings.ContainsAny(s.Title, "\r\n\f") {
			return fmt.Errorf("分组标题不能包含换行")
		}
		for _, step := range s.Steps {
			count++
			if err := checkID(step.ID); err != nil {
				return err
			}
			if _, err := parse(step.Expression); err != nil {
				return fmt.Errorf("%s：%w", step.Label, err)
			}
			if err := validatePrecision(step.Format, step.Precision); err != nil {
				return err
			}
			if len(step.Conversions) > 4 {
				return fmt.Errorf("单位换算步骤过多")
			}
			for _, c := range step.Conversions {
				if _, err := number(c.Factor); err != nil {
					return err
				}
				if err := validatePrecision(c.Format, c.Precision); err != nil {
					return err
				}
			}
			for _, v := range []string{step.Label, step.Symbolic, step.Unit} {
				if strings.ContainsAny(v, "\r\n\f") {
					return fmt.Errorf("计算项不能包含换行")
				}
				chars += len(v)
			}
			chars += len(step.Expression)
		}
	}
	if count < 1 || count > 200 || chars > 60000 {
		return fmt.Errorf("计算内容过多或为空，请分批导入")
	}
	return nil
}
func validatePrecision(format string, n int) error {
	if format != "decimal" && format != "significant" || n < 0 || n > 12 || format == "significant" && n < 1 {
		return fmt.Errorf("精度须为0–12位小数或1–12位有效数字")
	}
	return nil
}
func formatValue(v *big.Rat, format string, precision int) string {
	if format == "decimal" {
		return v.FloatString(precision)
	}
	if v.Sign() == 0 {
		if precision > 1 {
			return "0." + strings.Repeat("0", precision-1)
		}
		return "0"
	}
	f, _ := v.Float64()
	exp := int(math.Floor(math.Log10(math.Abs(f))))
	decimals := precision - 1 - exp
	if decimals >= 0 && decimals <= 12 {
		text := v.FloatString(decimals)
		rounded, _ := new(big.Rat).SetString(text)
		rv, _ := rounded.Float64()
		if rv != 0 && int(math.Floor(math.Log10(math.Abs(rv)))) > exp {
			return formatValue(rounded, format, precision)
		}
		return text
	}
	scale := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(absInt(exp))), nil))
	mantissa := new(big.Rat).Set(v)
	if exp >= 0 {
		mantissa.Quo(mantissa, scale)
	} else {
		mantissa.Mul(mantissa, scale)
	}
	text := mantissa.FloatString(precision - 1)
	rounded, _ := new(big.Rat).SetString(text)
	if new(big.Rat).Abs(rounded).Cmp(big.NewRat(10, 1)) >= 0 {
		rounded.Quo(rounded, big.NewRat(10, 1))
		text = rounded.FloatString(precision - 1)
		exp++
	}
	return fmt.Sprintf("%se%d", text, exp)
}
func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
func (p Plan) Calculate() (Result, error) {
	result := Result{Values: map[string]string{}}
	if err := p.Validate(); err != nil {
		return result, err
	}
	env := map[string]*big.Rat{}
	substitutions := map[string]string{}
	for _, v := range p.Variables {
		if v.Uncertain || v.Value == "" {
			return result, fmt.Errorf("请核对%s的数值（%s）", v.Label, v.Source)
		}
		n, err := number(v.Value)
		if err != nil {
			return result, err
		}
		env[v.ID] = n
		substitutions[v.ID] = v.Value
	}
	lines := []string{}
	for _, section := range p.Sections {
		if section.Title != "" {
			lines = append(lines, section.Title)
		}
		for _, step := range section.Steps {
			e, _ := parse(step.Expression)
			v, err := e.value(env)
			if err != nil {
				return result, fmt.Errorf("%s：%w", step.Label, err)
			}
			text := formatValue(v, step.Format, step.Precision)
			result.Values[step.ID] = text
			formula := step.Label + "="
			if step.Symbolic != "" {
				formula += step.Symbolic + "="
			}
			formula += e.mathText(substitutions) + "=" + scientific(text) + step.Unit
			for _, c := range step.Conversions {
				factor, _ := number(c.Factor)
				converted := new(big.Rat).Mul(v, factor)
				formula += "=" + scientific(formatValue(converted, c.Format, c.Precision)) + c.Unit
			}
			lines = append(lines, `\(`+formula+`\)`)
			env[step.ID] = v
			if step.UseRounded {
				env[step.ID], err = number(text)
				if err != nil {
					return result, err
				}
				substitutions[step.ID] = text
			} else {
				if v.IsInt() {
					substitutions[step.ID] = v.Num().String()
				} else {
					substitutions[step.ID] = `\frac{` + v.Num().String() + "}{" + v.Denom().String() + "}"
				}
			}
		}
	}
	if len(p.Issues) > 0 {
		return result, fmt.Errorf("请先核对识别疑问：%s", p.Issues[0])
	}
	result.Text = strings.Join(lines, "\n")
	return result, nil
}
