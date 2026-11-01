// Package calculation owns bounded arithmetic and editable calculation plans.
// Model output is data; expressions never execute Go, JavaScript or shell code.
package calculation

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type expression struct {
	kind, text string
	children   []*expression
}
type parser struct {
	s     []rune
	at, n int
}

func parse(s string) (*expression, error) {
	if len(s) > 2000 {
		return nil, fmt.Errorf("计算式过长")
	}
	p := parser{s: []rune(s)}
	e, err := p.expr(0, 0)
	p.space()
	if err == nil && p.at != len(p.s) {
		err = fmt.Errorf("计算式含不支持的符号：%c", p.s[p.at])
	}
	return e, err
}
func (p *parser) space() {
	for p.at < len(p.s) && unicode.IsSpace(p.s[p.at]) {
		p.at++
	}
}
func (p *parser) expr(minimum, depth int) (*expression, error) {
	p.n++
	if p.n > 300 || depth > 24 {
		return nil, fmt.Errorf("计算式过于复杂")
	}
	p.space()
	if p.at >= len(p.s) {
		return nil, fmt.Errorf("计算式不完整")
	}
	var left *expression
	r := p.s[p.at]
	switch {
	case r == '+' || r == '-':
		p.at++
		e, err := p.expr(3, depth+1)
		if err != nil {
			return nil, err
		}
		left = &expression{kind: "unary", text: string(r), children: []*expression{e}}
	case r == '(':
		p.at++
		e, err := p.expr(0, depth+1)
		if err != nil {
			return nil, err
		}
		p.space()
		if p.at >= len(p.s) || p.s[p.at] != ')' {
			return nil, fmt.Errorf("计算式括号未闭合")
		}
		p.at++
		left = e
	case r >= '0' && r <= '9' || r == '.':
		start := p.at
		for p.at < len(p.s) && (p.s[p.at] >= '0' && p.s[p.at] <= '9' || strings.ContainsRune(".eE", p.s[p.at]) || (p.s[p.at] == '+' || p.s[p.at] == '-') && p.at > start && (p.s[p.at-1] == 'e' || p.s[p.at-1] == 'E')) {
			p.at++
		}
		v := string(p.s[start:p.at])
		if _, err := number(v); err != nil {
			return nil, err
		}
		left = &expression{kind: "number", text: v}
	case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_':
		start := p.at
		p.at++
		for p.at < len(p.s) && (unicode.IsLetter(p.s[p.at]) || unicode.IsDigit(p.s[p.at]) || p.s[p.at] == '_') {
			p.at++
		}
		name := string(p.s[start:p.at])
		p.space()
		left = &expression{kind: "variable", text: name}
		if p.at < len(p.s) && p.s[p.at] == '(' {
			if name != "abs" && name != "sqrt" && name != "round" {
				return nil, fmt.Errorf("暂不支持函数%s", name)
			}
			p.at++
			arg, err := p.expr(0, depth+1)
			if err != nil {
				return nil, err
			}
			args := []*expression{arg}
			p.space()
			if name == "round" {
				if p.at >= len(p.s) || p.s[p.at] != ',' {
					return nil, fmt.Errorf("round需要小数位参数")
				}
				p.at++
				arg, err = p.expr(0, depth+1)
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
			}
			p.space()
			if p.at >= len(p.s) || p.s[p.at] != ')' {
				return nil, fmt.Errorf("函数括号未闭合")
			}
			p.at++
			left = &expression{kind: "function", text: name, children: args}
		}
	default:
		return nil, fmt.Errorf("计算式含不支持的符号：%c", r)
	}
	for {
		p.space()
		if p.at >= len(p.s) {
			break
		}
		op := p.s[p.at]
		prec := 0
		switch op {
		case '+', '-':
			prec = 1
		case '*', '/':
			prec = 2
		case '^':
			prec = 3
		}
		if prec == 0 || prec < minimum {
			break
		}
		p.at++
		next := prec + 1
		if op == '^' {
			next = prec
		}
		right, err := p.expr(next, depth+1)
		if err != nil {
			return nil, err
		}
		left = &expression{kind: "binary", text: string(op), children: []*expression{left, right}}
	}
	return left, nil
}

var decimalNumber = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]{1,3})?$`)

func number(s string) (*big.Rat, error) {
	if len(s) > 80 || !decimalNumber.MatchString(s) {
		return nil, fmt.Errorf("数值过长")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) || math.Abs(f) > 1e100 || f != 0 && math.Abs(f) < 1e-100 {
		return nil, fmt.Errorf("数值无效：%s", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || f == 0 && r.Sign() != 0 {
		return nil, fmt.Errorf("数值无效：%s", s)
	}
	return r, nil
}
func (e *expression) value(env map[string]*big.Rat) (*big.Rat, error) {
	if e.kind == "number" {
		return number(e.text)
	}
	if e.kind == "variable" {
		v, ok := env[e.text]
		if !ok {
			return nil, fmt.Errorf("缺少数值：%s", e.text)
		}
		return new(big.Rat).Set(v), nil
	}
	a, err := e.children[0].value(env)
	if err != nil {
		return nil, err
	}
	if e.kind == "unary" {
		if e.text == "-" {
			a.Neg(a)
		}
		return a, nil
	}
	if e.kind == "function" {
		switch e.text {
		case "abs":
			return a.Abs(a), nil
		case "sqrt":
			if a.Sign() < 0 {
				return nil, fmt.Errorf("负数不能开平方")
			}
			f, _ := a.Float64()
			return new(big.Rat).SetFloat64(math.Sqrt(f)), nil
		case "round":
			b, err := e.children[1].value(env)
			if err != nil {
				return nil, err
			}
			if !b.IsInt() || !b.Num().IsInt64() || b.Num().Int64() < 0 || b.Num().Int64() > 12 {
				return nil, fmt.Errorf("小数位须为0–12")
			}
			return number(a.FloatString(int(b.Num().Int64())))
		}
	}
	b, err := e.children[1].value(env)
	if err != nil {
		return nil, err
	}
	switch e.text {
	case "+":
		a.Add(a, b)
	case "-":
		a.Sub(a, b)
	case "*":
		a.Mul(a, b)
	case "/":
		if b.Sign() == 0 {
			return nil, fmt.Errorf("分母为0，请检查数据")
		}
		a.Quo(a, b)
	case "^":
		if !b.IsInt() || !b.Num().IsInt64() || b.Num().Int64() < -12 || b.Num().Int64() > 12 {
			return nil, fmt.Errorf("指数须为−12至12的整数")
		}
		n := b.Num().Int64()
		if n < 0 {
			if a.Sign() == 0 {
				return nil, fmt.Errorf("0不能取负指数")
			}
			a.Inv(a)
			n = -n
		}
		exponent := big.NewInt(n)
		a.SetFrac(new(big.Int).Exp(a.Num(), exponent, nil), new(big.Int).Exp(a.Denom(), exponent, nil))
	}
	if a.Num().BitLen() > 8192 || a.Denom().BitLen() > 8192 {
		return nil, fmt.Errorf("计算精度超出支持范围，请简化计算式")
	}
	f, _ := a.Float64()
	if math.IsInf(f, 0) || math.IsNaN(f) || math.Abs(f) > 1e100 || a.Sign() != 0 && math.Abs(f) < 1e-100 {
		return nil, fmt.Errorf("计算结果超出支持范围")
	}
	return a, nil
}
func (e *expression) mathText(values map[string]string) string {
	if e.kind == "number" {
		return scientific(e.text)
	}
	if e.kind == "variable" {
		v := values[e.text]
		if strings.HasPrefix(v, "-") {
			return "(" + scientific(v) + ")"
		}
		return scientific(v)
	}
	a := e.children[0].mathText(values)
	if e.kind == "unary" {
		return e.text + "(" + a + ")"
	}
	if e.kind == "function" {
		switch e.text {
		case "abs":
			if e.children[0].kind == "variable" && strings.HasPrefix(values[e.children[0].text], "-") {
				a = strings.TrimSuffix(strings.TrimPrefix(a, "("), ")")
			}
			return "|" + a + "|"
		case "sqrt":
			return `\sqrt{` + a + "}"
		case "round":
			env := map[string]*big.Rat{}
			for id, text := range values {
				if n, err := number(text); err == nil {
					env[id] = n
				}
			}
			if rounded, err := e.value(env); err == nil {
				n, _ := e.children[1].value(env)
				return rounded.FloatString(int(n.Num().Int64()))
			}
			return "round(" + a + "," + e.children[1].mathText(values) + ")"
		}
	}
	b := e.children[1].mathText(values)
	switch e.text {
	case "/":
		return `\frac{` + a + "}{" + b + "}"
	case "^":
		return "(" + a + ")^{" + b + "}"
	case "*":
		if e.children[0].text == "+" || e.children[0].text == "-" {
			a = "(" + a + ")"
		}
		if e.children[1].text == "+" || e.children[1].text == "-" {
			b = "(" + b + ")"
		}
		return a + "×" + b
	case "-":
		if e.children[1].kind == "binary" && (e.children[1].text == "+" || e.children[1].text == "-") {
			b = "(" + b + ")"
		}
	}
	return a + e.text + b
}
func scientific(s string) string {
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		return s[:i] + "×10^{" + strings.TrimPrefix(s[i+1:], "+") + "}"
	}
	return s
}
