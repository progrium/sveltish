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
		for _, node := range flatten(c.HTML) {
			s.Stmt("let", node.Tag)
		}
		s.Line("")
		s.Stmt("return", func(s *js.Source) {
			s.Stmt("c()", func(s *js.Source) {
				for _, node := range flatten(c.HTML) {
					s.Stmt(node.Tag, "=", fmt.Sprintf(`element("%s")`, node.Tag))
					if node.Text != "" {
						s.Stmt(s.Chain(node.Tag, "textContent"), "=", fmt.Sprintf("`%s`", strings.Replace(node.Text, "{", "${", -1)))
					}
				}
			}, ",")
			s.Stmt("m(target, anchor)", func(s *js.Source) {
				for _, node := range c.HTML.Children {
					s.Stmt(s.Call("insert", "target", node.Tag, "anchor"))
					walk(node, func(nn *Node) bool {
						if nn != node {
							s.Stmt(s.Call("append", nn.Parent.Tag, nn.Tag))
						}
						return true
					})
				}
			}, ",")
			s.Line("p: noop,")
			s.Line("i: noop,")
			s.Line("o: noop,")
			s.Stmt("d(detaching)", func(s *js.Source) {
				for _, node := range c.HTML.Children {
					s.Stmt("if (detaching)", s.Call("detach", node.Tag))
				}
			})

		})
	})
	s.Line("")
	s.Line(c.JS.Children[0].Text)
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
