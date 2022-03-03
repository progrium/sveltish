package sveltish

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/progrium/sveltish/internal/html"
	"github.com/progrium/sveltish/internal/js"
)

func GenerateJS(c *Component) ([]byte, error) {
	nrw := js.NewVarNameRewriter(c.JS, func(i int, name string, _ js.Var, _ []byte) []byte {
		return generateCtxLookup(i, name)
	})
	arw := js.NewAssignmentRewriter(c.JS, func(i int, _ string, _ js.Var, data []byte) []byte {
		newData := [][]byte{}
		newData = append(newData, []byte(fmt.Sprintf("$$invalidate(%d, ", i)))
		newData = append(newData, data)
		newData = append(newData, []byte(")"))
		return bytes.Join(newData, nil)
	})

	decStmts := []string{}
	setStmts := []string{}
	mntStmts := []string{}
	lsnStmts := []string{}
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
				fmt.Sprintf("detach(%s)", nv.name),
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
			valContent, info := node.RewriteJs(nrw)

			valName := fmt.Sprintf("%s_value", nv.name)
			decStmts = append(
				decStmts,
				fmt.Sprintf("let %s = %s", valName, valContent),
			)
			setStmts = append(
				setStmts,
				fmt.Sprintf("%s = text(%s)", nv.name, valName),
			)

			if valDirty := info.Dirty(); valDirty != 0 {
				updStmts = append(
					updStmts,
					fmt.Sprintf(
						"if (dirty & /*%s*/ %d && %s !== (%s = %s)) set_data(%s, %s)",
						strings.Join(info.Names(), " "),
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
				attContent, info := attr.RewriteJs(nrw)

				dir, exists := attr.Dir()
				if exists {
					if name := attr.Name(); name != "on" {
						return nil, errors.New("Invaild attribute with directive, " + name + ":" + dir)
					}

					lsnStmts = append(
						lsnStmts,
						fmt.Sprintf("listen(%s, '%s', %s)", nv.name, dir, attContent),
					)
					continue
				}

				attName := generateAttrName(nv.name, attr)
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
				if attrDirty := info.Dirty(); attrDirty != 0 {
					updStmts = append(
						updStmts,
						fmt.Sprintf(
							"if (dirty & /*%s*/ %d) %s",
							strings.Join(info.Names(), " "),
							attrDirty,
							setStmt,
						),
					)
				}
			}
		}
	}

	data, info := c.JS.RewriteForInstance(
		arw,
		func(wrapUpds func(js.WrapUpdFn) []byte) []byte {
			wrpData := [][]byte{}
			wrpData = append(wrpData, []byte("\n$$self.$$.update = () => {\n"))
			wrpData = append(
				wrpData,
				wrapUpds(func(info *js.VarsInfo, updData []byte) []byte {
					return []byte(fmt.Sprintf(
						"if ($$self.$$.dirty & /*%s*/ %d) {%s\n}\n",
						strings.Join(info.Names(), " "),
						info.Dirty(),
						updData,
					))
				}),
			)
			wrpData = append(wrpData, []byte("}\n"))

			return bytes.Join(wrpData, nil)
		},
	)
	instBody := string(data)
	instReturns := info.Names()

	s := &js.Source{}
	s.Stmt(`import {
  SvelteComponent,
  append,
  detach,
  element,
  text,
  space,
  attr,
  listen,
  init,
  insert,
  noop,
  safe_not_equal,
  set_data,
  run_all
} from`, s.Str("./runtime"))
	s.Line("")
	s.Func("create_fragment", []string{"ctx"}, func(s *js.Source) {
		for _, decStmt := range decStmts {
			s.Stmt(decStmt)
		}
		s.Stmt("let mounted")
		s.Stmt("let dispose")

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

				s.Stmt("if(!mounted)", func(s *js.Source) {
					s.Line("dispose = [")
					for _, lsnStmt := range lsnStmts {
						s.Line("  " + lsnStmt + ",")
					}
					s.Line("];")

					s.Stmt("mounted = true")
				})
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
					s.Stmt("if (detaching)", detStmt)
				}

				s.Stmt("mounted = false")
				s.Stmt("run_all(dispose)")
			})

		})
	})
	s.Line("")
	if c.JS != nil {
		s.Func("instance", []string{"$$self", "$$props", "$$invalidate"}, func(s *js.Source) {
			s.Line(instBody)
			s.Stmt("return [" + strings.Join(instReturns, ", ") + "]")
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

func generateCtxLookup(i int, name string) []byte {
	return []byte(fmt.Sprintf("/* %s */ ctx[%d]", name, i))
}
