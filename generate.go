package sveltish

import (
	"fmt"
	"strings"

	"github.com/progrium/sveltish/internal/html"
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
		html.Walk(c.HTML, func(n html.Node, _ []html.NodeContainer) bool {
			switch node := n.(type) {
			case *html.ElNode:
				s.Stmt("let", node.Tag)
			}
			return true
		})
		s.Line("")
		s.Stmt("return", func(s *js.Source) {
			s.Stmt("c()", func(s *js.Source) {
				html.Walk(c.HTML, func(n html.Node, _ []html.NodeContainer) bool {
					switch node := n.(type) {
					case *html.ElNode:
						s.Stmt(node.Tag, "=", fmt.Sprintf(`element("%s")`, node.Tag))
						if len(node.ChildNodes) == 1 {
							switch child := node.ChildNodes[0].(type) {
							case *html.TxtNode:
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
				html.Walk(c.HTML, func(n html.Node, ps []html.NodeContainer) bool {
					switch node := n.(type) {
					case *html.ElNode:
						s.Stmt(s.Call("insert", "target", node.Tag, "anchor"))

						if len(ps) > 1 {
							switch parent := ps[0].(type) {
							case *html.ElNode:
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
				html.Walk(c.HTML, func(n html.Node, _ []html.NodeContainer) bool {
					switch node := n.(type) {
					case *html.ElNode:
						s.Stmt("if (detaching)", s.Call("detach", node.Tag))
					}
					return true
				})
			})

		})
	})
	s.Line("")
	for _, jsEl := range c.JS {
		s.Line(jsEl.Content)
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
