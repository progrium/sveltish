package sveltish

import (
	"fmt"
	"strings"

	"github.com/progrium/sveltish/internal/html"
	"github.com/progrium/sveltish/internal/js"
)

func GenerateJS(c *Component) ([]byte, error) {
	rootVars := c.JS.RootVars()

	decStmts := []string{}
	setStmts := []string{}
	mntStmts := []string{}
	detStmts := []string{}
	updStmts := []string{}
	for _, nv := range c.HTML {
		decStmts = append(decStmts, fmt.Sprintf("let %s", nv.name))
		if nv.hasParent {
			mntStmts = append(
				mntStmts,
				fmt.Sprintf(
					"append(%s, %s)",
					nv.parentName,
					nv.name,
				),
			)
		} else {
			mntStmts = append(
				mntStmts,
				fmt.Sprintf(
					"insert(target, %s, anchor)",
					nv.name,
				),
			)
			detStmts = append(
				detStmts,
				fmt.Sprintf("if (detaching) detach(%s)", nv.name),
			)
		}

		switch node := nv.node.(type) {
		case *html.ElNode:
			setStmts = append(
				setStmts,
				fmt.Sprintf(`%s = element("%s")`, nv.name, node.Tag()),
			)
		case *html.LeafElNode:
			setStmts = append(
				setStmts,
				fmt.Sprintf(`%s = element("%s")`, nv.name, node.Tag()),
			)
		case *html.TxtNode:
			if html.IsContentWhiteSpace(node) {
				setStmts = append(
					setStmts,
					fmt.Sprintf("%s = space()", nv.name),
				)
			} else {
				setStmts = append(
					setStmts,
					fmt.Sprintf(`%s = text("%s")`, nv.name, node.Content()),
				)
			}
		case *html.ExprNode:
			valName := fmt.Sprintf("%s_value", nv.name)
			varNames := []string{}
			valDirty := 0
			valContent := node.JsContent(rootVars, func(i int, jsNV *js.NamedVar, _ []byte) []byte {
				varNames = append(varNames, jsNV.Name)
				valDirty += 1 << i

				return []byte(generateCtxLookup(i, jsNV))
			})
			setVal := fmt.Sprintf("%s = %s", valName, valContent)

			decStmts = append(
				decStmts,
				fmt.Sprintf("let %s", setVal),
			)
			setStmts = append(
				setStmts,
				fmt.Sprintf("%s = text(%s)", nv.name, valName),
			)
			if valDirty != 0 {
				updStmts = append(
					updStmts,
					fmt.Sprintf(
						"if (dirty & /*%s*/ %d && %s !== (%s = %s)) set_data(%s, %s)",
						strings.Join(varNames, " "),
						valDirty,
						valName,
						valName,
						valContent,
						nv.name,
						valName,
					),
				)
			}
		}

		if el, ok := nv.node.(html.Element); ok {
			for _, attr := range el.Attrs() {
				attName := generateAttrName(nv.name, attr)
				varNames := []string{}
				attrDirty := 0
				attContent := attr.JsContent(rootVars, func(i int, jsNV *js.NamedVar, _ []byte) []byte {
					varNames = append(varNames, jsNV.Name)
					attrDirty += 1 << i

					return []byte(generateCtxLookup(i, jsNV))
				})

				decStmts = append(
					decStmts,
					fmt.Sprintf("let %s", attName),
				)
				setStmt := fmt.Sprintf(
					"attr(%s, '%s', %s = %s)",
					nv.name,
					attr.Name(),
					attName,
					attContent,
				)
				setStmts = append(
					setStmts,
					setStmt,
				)
				if attrDirty != 0 {
					updStmts = append(
						updStmts,
						fmt.Sprintf(
							"if (dirty & /*%s*/ %d) %s",
							strings.Join(varNames, " "),
							attrDirty,
							setStmt,
						),
					)
				}
			}
		}
	}

	s := &js.Source{}
	s.Stmt(`import {
  SvelteComponent,
  append,
  detach,
  element,
  text,
  space,
  attr,
  init,
  insert,
  noop,
  safe_not_equal,
  set_data
} from`, s.Str("./runtime"))
	s.Line("")
	s.Func("create_fragment", []string{"ctx"}, func(s *js.Source) {
		for _, decStmt := range decStmts {
			s.Stmt(decStmt)
		}
		s.Line("")
		s.Stmt("return", func(s *js.Source) {
			s.Stmt("c()", func(s *js.Source) {
				for _, setStmt := range setStmts {
					s.Stmt(setStmt)
				}
			}, ",")
			s.Stmt("m(target, anchor)", func(s *js.Source) {
				for _, mntStmt := range mntStmts {
					s.Stmt(mntStmt)
				}
			}, ",")
			s.Stmt("p(ctx, [dirty])", func(s *js.Source) {
				for _, updStmt := range updStmts {
					s.Stmt(updStmt)
				}
			}, ",")
			s.Line("i: noop,")
			s.Line("o: noop,")
			s.Stmt("d(detaching)", func(s *js.Source) {
				for _, detStmt := range detStmts {
					s.Stmt(detStmt)
				}
			})

		})
	})
	s.Line("")
	if c.JS != nil {
		s.Func("instance", []string{"$$self", "$$props", "$$invalidate"}, func(s *js.Source) {
			c.JS.WrapAssignments(func(i int, _ *js.NamedVar) (string, string) {
				return fmt.Sprintf("$$invalidate(%d, ", i), ")"
			})
			s.Line(c.JS.Js())

			names := []string{}
			for _, v := range rootVars {
				names = append(names, v.Name)
			}
			s.Stmt("return [" + strings.Join(names, ", ") + "]")
		})
	}
	s.Line("")
	s.Stmt("class", c.Name, "extends SvelteComponent", func(s *js.Source) {
		s.Stmt("constructor(options)", func(s *js.Source) {
			s.Stmt("super()")
			s.Stmt("init(this, options, instance, create_fragment, safe_not_equal, {})")
		})
	})
	s.Line("")
	s.Stmt("export default", c.Name)
	fmt.Println(s.String())
	return s.Bytes(), nil
}

func generateAttrName(nodeName string, attr html.Attr) string {
	attrName := strings.ReplaceAll(attr.Name(), "-", "_")

	return fmt.Sprintf("%s_%s_value", nodeName, attrName)
}

func generateCtxLookup(i int, jsNV *js.NamedVar) string {
	return fmt.Sprintf("/* %s */ ctx[%d]", jsNV.Name, i)
}
