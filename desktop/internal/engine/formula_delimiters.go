package engine

import "fmt"

// Accept a complete delimited span after it has been joined to surrounding
// prose. The body keeps the delimiters in its source and selection cell while
// the shared parser renders only the enclosed mathematics.
func (p *formulaParser) delimited(depth int) (imageMathBox, error) {
	closeFor := func(open rune) rune {
		if open == '[' {
			return ']'
		}
		return ')'
	}
	start := p.pos + 1
	stack := []rune{closeFor(p.input[p.pos])}
	for i := start; i+1 < len(p.input); i++ {
		if p.input[i] != '\\' {
			continue
		}
		next := p.input[i+1]
		if next == '\\' {
			i++
			continue
		}
		if next == '[' || next == '(' {
			stack = append(stack, closeFor(next))
			i++
			continue
		}
		if next != ']' && next != ')' {
			continue
		}
		if next != stack[len(stack)-1] {
			return imageMathBox{}, fmt.Errorf("公式定界符不匹配，请检查\\(...\\)或\\[...\\]")
		}
		stack = stack[:len(stack)-1]
		if len(stack) > 0 {
			i++
			continue
		}
		inner := *p
		inner.input, inner.pos = p.input[start:i], 0
		inner.variation = p.variation.at(start)
		b, err := inner.sequence(depth+1, 0)
		p.pos, p.nodes = i+2, inner.nodes
		p.hasFraction = p.hasFraction || inner.hasFraction
		return b, err
	}
	return imageMathBox{}, fmt.Errorf("公式未闭合，请补齐\\%c", stack[len(stack)-1])
}
