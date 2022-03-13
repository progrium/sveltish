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
	sg, err := newScriptGenerator(c)
	if err != nil {
		return nil, err
	}

	s := sg.Source()
	return s.Bytes(), nil
}

type stmtType int

const (
	dec stmtType = iota
	set
	mnt
	lsn
	det
	upd
)

type scriptGenerator struct {
	name        string
	stmts       map[stmtType][]string
	instBody    string
	instReturns []string
}

func newScriptGenerator(c *Component) (*scriptGenerator, error) {
	sg := &scriptGenerator{
		name:  c.Name,
		stmts: map[stmtType][]string{},
	}

	nrw := js.NewVarNameRewriter(c.JS, func(i int, name string, _ js.Var, _ []byte) []byte {
		return []byte(fmt.Sprintf("/* %s */ ctx[%d]", name, i))
	})

	evtNames := map[string]int{}
	for _, nv := range c.HTML {
		sg.insertf(
			dec,
			"let %s",
			nv.name,
		)
		if nv.hasParent {
			sg.insertf(
				mnt,
				"append(%s, %s)",
				nv.parentName,
				nv.name,
			)
		} else {
			sg.insertf(
				mnt,
				"insert(target, %s, anchor)",
				nv.name,
			)
			sg.insertf(
				det,
				"if (detaching) detach(%s)",
				nv.name,
			)
		}

		switch node := nv.node.(type) {
		case *html.ElNode:
			sg.insertf(
				set,
				`%s = element("%s")`,
				nv.name,
				node.Tag(),
			)
		case *html.LeafElNode:
			sg.insertf(
				set,
				`%s = element("%s")`,
				nv.name,
				node.Tag(),
			)
		case *html.TxtNode:
			if html.IsContentWhiteSpace(node) {
				sg.insertf(
					set,
					"%s = space()",
					nv.name,
				)
			} else {
				sg.insertf(
					set,
					`%s = text("%s")`,
					nv.name,
					node.Content(),
				)
			}
		case *html.ExprNode:
			valContent, info := node.RewriteJs(nrw)

			valName := fmt.Sprintf("%s_value", nv.name)
			sg.insertf(
				dec,
				"let %s = %s",
				valName,
				valContent,
			)
			sg.insertf(
				set,
				"%s = text(%s)",
				nv.name,
				valName,
			)

			if valDirty := info.Dirty(); valDirty != 0 {
				sg.insertf(
					upd,
					"if (dirty & /*%s*/ %d && %s !== (%s = %s)) set_data(%s, %s)",
					strings.Join(info.Names(), " "),
					valDirty,
					valName,
					valName,
					valContent,
					nv.name,
					valName,
				)
			}
		}

		if el, ok := nv.node.(html.Element); ok {
			for _, attr := range el.Attrs() {
				if html.IsInlineEvt(attr) {
					attContent, _ := attr.RewriteJs(nil)
					dir, _ := attr.Dir()

					varName := dir + "_handler"
					if count, exists := evtNames[varName]; exists {
						count += 1
						evtNames[varName] = count

						varName += fmt.Sprintf("_%d", count)
					} else {
						evtNames[varName] = 0
					}

					c.JS.AppendConst(varName, string(attContent))
					varCtx, _ := nrw.Rewrite([]byte(varName))
					sg.insertf(
						lsn,
						"listen(%s, '%s', %s)",
						nv.name,
						dir,
						varCtx,
					)
					continue
				}

				attContent, info := attr.RewriteJs(nrw)
				dir, exists := attr.Dir()
				if exists {
					if name := attr.Name(); name != "on" {
						return sg, errors.New("Invaild attribute with directive, " + name + ":" + dir)
					}

					sg.insertf(
						lsn,
						"listen(%s, '%s', %s)",
						nv.name,
						dir,
						attContent,
					)
					continue
				}

				attName := fmt.Sprintf(
					"%s_%s_value",
					nv.name,
					strings.ReplaceAll(attr.Name(), "-", "_"),
				)
				sg.insertf(
					dec,
					"let %s",
					attName,
				)

				setAttrStmt := fmt.Sprintf(
					"attr(%s, '%s', %s = %s)",
					nv.name,
					attr.Name(),
					attName,
					attContent,
				)
				sg.insert(set, setAttrStmt)

				if attrDirty := info.Dirty(); attrDirty != 0 {
					sg.insertf(
						upd,
						"if (dirty & /*%s*/ %d) %s",
						strings.Join(info.Names(), " "),
						attrDirty,
						setAttrStmt,
					)
				}
			}
		}
	}

	if c.JS == nil {
		return sg, nil
	}

	arw := js.NewAssignmentRewriter(c.JS, func(i int, _ string, _ js.Var, data []byte) []byte {
		newData := [][]byte{}
		newData = append(newData, []byte(fmt.Sprintf("$$invalidate(%d, ", i)))
		newData = append(newData, data)
		newData = append(newData, []byte(")"))
		return bytes.Join(newData, nil)
	})
	data, info := c.JS.RewriteForInstance(
		arw,
		func(wrapUpds func(js.WrapUpdFn) []byte) []byte {
			wrpData := [][]byte{}
			wrpData = append(wrpData, []byte("\n$$self.$$.update = () => {\n"))
			wrpData = append(
				wrpData,
				wrapUpds(func(labelInfo *js.VarsInfo, updData []byte) []byte {
					return []byte(fmt.Sprintf(
						"if ($$self.$$.dirty & /*%s*/ %d) {%s\n}\n",
						strings.Join(labelInfo.Names(), " "),
						labelInfo.Dirty(),
						updData,
					))
				}),
			)
			wrpData = append(wrpData, []byte("};\n"))

			return bytes.Join(wrpData, nil)
		},
	)
	sg.instBody = string(data)
	sg.instReturns = info.Names()

	return sg, nil
}

func (sg *scriptGenerator) insert(st stmtType, stmt string) {
	currStmts, exists := sg.stmts[st]
	if !exists {
		sg.stmts[st] = []string{stmt}
		return
	}

	sg.stmts[st] = append(currStmts, stmt)
}

func (sg *scriptGenerator) insertf(st stmtType, format string, a ...interface{}) {
	sg.insert(st, fmt.Sprintf(format, a...))
}

func (sg *scriptGenerator) printStmts(s *js.Source, st stmtType) {
	currStmts, exists := sg.stmts[st]
	if !exists {
		return
	}

	for _, stmt := range currStmts {
		s.Stmt(stmt)
	}
}

func (sg *scriptGenerator) hasInst() bool {
	return sg.instBody != "" || len(sg.instReturns) != 0
}

func (sg *scriptGenerator) printInst(s *js.Source) {
	s.Line(sg.instBody)
	s.Stmt(fmt.Sprintf(
		"return [%s]",
		strings.Join(sg.instReturns, ", "),
	))
}

func (sg *scriptGenerator) Source() *js.Source {
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
		sg.printStmts(s, dec)
		s.Stmt("let mounted")
		s.Stmt("let dispose")

		s.Line("")
		s.Stmt("return", func(s *js.Source) {
			s.Stmt("c()", func(s *js.Source) {
				sg.printStmts(s, set)
			}, ",")
			s.Stmt("m(target, anchor)", func(s *js.Source) {
				sg.printStmts(s, mnt)

				s.Stmt("if(!mounted)", func(s *js.Source) {
					s.Line("dispose = [")
					if lsnStmts, exists := sg.stmts[lsn]; exists {
						for _, lsnStmt := range lsnStmts {
							s.Line("  " + lsnStmt + ",")
						}
					}
					s.Line("];")
					s.Line("")
					s.Stmt("mounted = true")
				})
			}, ",")
			s.Stmt("p(ctx, [dirty])", func(s *js.Source) {
				sg.printStmts(s, upd)
			}, ",")
			s.Line("i: noop,")
			s.Line("o: noop,")
			s.Stmt("d(detaching)", func(s *js.Source) {
				sg.printStmts(s, det)
				s.Line("")
				s.Stmt("mounted = false")
				s.Stmt("run_all(dispose)")
			}, ",")
		}, ";")
	})
	s.Line("")
	if sg.hasInst() {
		s.Func("instance", []string{"$$self", "$$props", "$$invalidate"}, func(s *js.Source) {
			sg.printInst(s)
		})
	}
	s.Line("")
	s.Stmt("class", sg.name, "extends SvelteComponent", func(s *js.Source) {
		s.Stmt("constructor(options)", func(s *js.Source) {
			s.Stmt("super()")
			s.Stmt("init(this, options, instance, create_fragment, safe_not_equal, {})")
		})
	})
	s.Line("")
	s.Stmt("export default", sg.name)

	//
	fmt.Println(s.String())
	//*/

	return s
}
