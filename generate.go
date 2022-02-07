package sveltish

import (
	"fmt"

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
  text,
  space,
  init,
  insert,
  noop,
  safe_not_equal
} from`, s.Str("./runtime"))
	s.Line("")
	s.Func("create_fragment", []string{"ctx"}, func(s *js.Source) {
		for _, nv := range c.HTML {
			s.Stmt("let", nv.name)
			if exn, ok := nv.node.(*html.ExprNode); ok {
				s.Stmt("let", fmt.Sprintf("%s_value", nv.name), "=", exn.JsContent())
			}
		}
		s.Line("")
		s.Stmt("return", func(s *js.Source) {
			s.Stmt("c()", func(s *js.Source) {
				for _, nv := range c.HTML {
					switch node := nv.node.(type) {
					case *html.ElNode:
						s.Stmt(nv.name, "=", fmt.Sprintf(`element("%s")`, node.Tag()))
					case *html.LeafElNode:
						s.Stmt(nv.name, "=", fmt.Sprintf(`element("%s")`, node.Tag()))
					case *html.TxtNode:
						if html.IsContentWhiteSpace(node) {
							s.Stmt(nv.name, "=", "space()")
						} else {
							s.Stmt(nv.name, "=", fmt.Sprintf(`text("%s")`, node.Content()))
						}
					case *html.ExprNode:
						s.Stmt(nv.name, "=", fmt.Sprintf("text(%s_value)", nv.name))
					}
				}
			}, ",")
			s.Stmt("m(target, anchor)", func(s *js.Source) {
				for _, nv := range c.HTML {
					if nv.hasParent {
						s.Stmt(s.Call("append", nv.parentName, nv.name))
					} else {
						s.Stmt(s.Call("insert", "target", nv.name, "anchor"))
					}
				}
			}, ",")
			s.Line("p: noop,")
			s.Line("i: noop,")
			s.Line("o: noop,")
			s.Stmt("d(detaching)", func(s *js.Source) {
				for _, nv := range c.HTML {
					if nv.hasParent {
						continue
					}

					s.Stmt("if (detaching)", s.Call("detach", nv.name))
				}
			})

		})
	})
	s.Line("")
	for _, jsEl := range c.JS {
		s.Line(jsEl.Content())
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
