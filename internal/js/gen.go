package js

import (
	"fmt"
	"strings"
)

type Source struct {
	stack [][]string
	stmts []string
}

func (s *Source) String() string {
	return strings.Join(s.stmts, "\n")
}

func (s *Source) Bytes() []byte {
	return []byte(s.String())
}

func (s *Source) Block(fn func(*Source)) string {
	s.stack = append(s.stack, s.stmts)
	s.stmts = []string{}
	fn(s)
	stmts := s.stmts
	s.stmts, s.stack = s.stack[len(s.stack)-1], s.stack[:len(s.stack)-1]
	return strings.Join(stmts, "\n")
}

func (s *Source) Pop() (stmt string) {
	stmt, s.stmts = s.stmts[len(s.stmts)-1], s.stmts[:len(s.stmts)-1]
	return
}

func (s *Source) Line(str string) {
	s.stmts = append(s.stmts, s.Indent()+str)
}

func (s *Source) Stmt(args ...interface{}) {
	var parts []string
	semi := true
	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			parts = append(parts, a)
		case func(*Source):
			parts = append(parts, fmt.Sprintf("{\n%s\n%s}", s.Block(a), s.Indent()))
			semi = false
		}
	}
	line := strings.Join(parts, " ")
	if semi {
		s.Line(line + ";")
	} else {
		s.Line(line)
	}
}

func (s *Source) Func(name string, args []string, block func(*Source)) {
	s.Stmt("function", fmt.Sprintf("%s%s", name, s.signature(args)), block)
}

func (s *Source) signature(args []string) string {
	return fmt.Sprintf("(%s)", strings.Join(args, ", "))
}

func (s *Source) Indent() string {
	indent := ""
	if len(s.stack) > 0 {
		for i := 0; i < len(s.stack); i++ {
			indent += "  "
		}
	}
	return indent
}

func (s *Source) Call(name string, args ...string) string {
	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
}

func (ss *Source) Comment(s string) {
	ss.Line(fmt.Sprintf("// %s", s))
}

func (s *Source) Str(v string) string {
	return fmt.Sprintf(`"%s"`, v)
}

func (s *Source) Chain(args ...string) string {
	return strings.Join(args, ".")
}
