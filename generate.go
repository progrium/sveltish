package sveltish

import (
	"fmt"
	"strings"

	"github.com/progrium/sveltish/internal/js"
)

func GenerateJS(c *Component) ([]byte, error) {
	s := &js.Source{}
	s.Stmt(`import {
  SvelteComponent,
  append,
  detach,
  element,
  init,
  insert,
  noop,
  safe_not_equal
} from`, s.Str("./runtime"))
	s.Line("")
	s.Func("create_fragment", []string{"ctx"}, func(s *js.Source) {
		WalkHTML(c.HTML, func(n HTMLNode, _ []HTMLNode) bool {
			switch node := n.(type) {
			case *ElNode:
				s.Stmt("let", node.Tag)
			}
			return true
		})
		s.Line("")
		s.Stmt("return", func(s *js.Source) {
			s.Stmt("c()", func(s *js.Source) {
				WalkHTML(c.HTML, func(n HTMLNode, _ []HTMLNode) bool {
					switch node := n.(type) {
					case *ElNode:
						s.Stmt(node.Tag, "=", fmt.Sprintf(`element("%s")`, node.Tag))
						if len(node.Els) == 1 {
							switch child := node.Els[0].(type) {
							case *TxtNode:
								s.Stmt(
									s.Chain(node.Tag, "textContent"),
									"=",
									fmt.Sprintf("`%s`", strings.Replace(child.Content, "{", "${", -1)),
								)
								return false
							}
						}
					}
					return true
				})
			}, ",")
			s.Stmt("m(target, anchor)", func(s *js.Source) {
				WalkHTML(c.HTML, func(n HTMLNode, ps []HTMLNode) bool {
					switch node := n.(type) {
					case *ElNode:
						s.Stmt(s.Call("insert", "target", node.Tag, "anchor"))

						if len(ps) > 1 {
							switch parent := ps[0].(type) {
							case *ElNode:
								s.Stmt(s.Call("append", parent.Tag, node.Tag))
							}
						}
					}
					return true
				})
			}, ",")
			s.Line("p: noop,")
			s.Line("i: noop,")
			s.Line("o: noop,")
			s.Stmt("d(detaching)", func(s *js.Source) {
				WalkHTML(c.HTML, func(n HTMLNode, _ []HTMLNode) bool {
					switch node := n.(type) {
					case *ElNode:
						s.Stmt("if (detaching)", s.Call("detach", node.Tag))
					}
					return true
				})
			})

		})
	})
	s.Line("")
	WalkHTML(c.JS, func(n HTMLNode, _ []HTMLNode) bool {
		switch node := n.(type) {
		case *LeafElNode:
			s.Line(node.Content)
		}
		return true
	})
	if len(c.JS.Roots) > 0 {

	}
	s.Line("")
	s.Stmt("class", c.Name, "extends SvelteComponent", func(s *js.Source) {
		s.Stmt("constructor(options)", func(s *js.Source) {
			s.Stmt("super()")
			s.Stmt("init(this, options, null, create_fragment, safe_not_equal, {})")
		})
	})
	s.Line("")
	s.Stmt("export default", c.Name)
	fmt.Println(s.String())
	return s.Bytes(), nil
}
