package commands

import (
	"strings"
)

type completionType int

const (
	NONE completionType = iota
	COMMAND
	OPERAND
	SHORT_OPTION
	OPTION_ARGUMENT
)

type parser struct {
	tokens []string
	optind int
	spec   string
	space  bool
	kind   completionType
	flag   string
	arg    string
	err    error
}

func newParser(cmd, spec string, spaceTerminated bool) (*parser, error) {
	args, err := splitCmd(cmd)
	if err != nil {
		return nil, err
	}

	p := &parser{
		tokens: args,
		optind: 0,
		spec:   spec,
		space:  spaceTerminated,
		kind:   NONE,
		flag:   "",
		arg:    "",
		err:    nil,
	}

	state := command
	for state != nil {
		state = state(p)
	}

	return p, p.err
}

func (p *parser) empty() bool {
	return len(p.tokens) == 0
}

func (p *parser) peek() string {
	return p.tokens[0]
}

func (p *parser) advance() string {
	if p.empty() {
		return ""
	}
	tok := p.tokens[0]
	p.tokens = p.tokens[1:]
	p.optind++
	return tok
}

func (p *parser) set(t completionType) {
	p.kind = t
}

func (p *parser) hasArgument() bool {
	n := len(p.flag)
	if n > 0 {
		s := string(p.flag[n-1]) + ":"
		return strings.Contains(p.spec, s)
	}
	return false
}

type stateFn func(*parser) stateFn

func command(p *parser) stateFn {
	p.set(COMMAND)
	p.advance()
	return peek(p)
}

func peek(p *parser) stateFn {
	if p.empty() {
		if p.space {
			return operand
		}
		return nil
	}
	if p.spec == "" {
		return operand
	}
	s := p.peek()
	switch {
	case s == "--":
		p.advance()
	case strings.HasPrefix(s, "-"):
		return short_option
	}
	return operand
}

func short_option(p *parser) stateFn {
	p.set(SHORT_OPTION)
	tok := p.advance()
	p.flag = tok[1:]
	if p.hasArgument() {
		return option_argument
	}
	return peek(p)
}

func option_argument(p *parser) stateFn {
	p.set(OPTION_ARGUMENT)
	p.arg = p.advance()
	if p.empty() && len(p.arg) == 0 {
		return nil
	}
	return peek(p)
}

func operand(p *parser) stateFn {
	p.set(OPERAND)
	return nil
}
