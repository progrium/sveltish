package html

import (
	"bytes"
	"errors"
	"strings"
	"unicode"

	"github.com/progrium/sveltish/internal/js"
)

// An attr represents an attribute on an html element.
type Attr interface {
	Name() string
	RewriteJs(rw js.VarRewriter) ([]byte, js.RewriteInfo)

	//TODO, add for attributes with ":..." directives
	//Dir() (string, bool)
}

func newAttr(data []byte) (Attr, error) {
	prts := bytes.SplitN(data, []byte("="), 2)

	namePrt := stripInitWhiteSpace(prts[0])
	if len(prts) == 1 {
		return &staticAttr{
			name:    string(namePrt),
			content: "",
		}, nil
	}

	valuePrt := stripQuotes(prts[1])
	index := indexStartExpr(valuePrt)
	if index == -1 {
		return &staticAttr{
			name:    string(namePrt),
			content: string(valuePrt),
		}, nil
	}

	tmpl := []string{string(valuePrt[:index])}
	exprs := []string{}

	for remaingPrt := valuePrt[index:]; len(remaingPrt) > 0; {
		var nextIndex int
		if len(tmpl) == len(exprs) {
			nextIndex = indexStartExpr(remaingPrt)
			if nextIndex == -1 {
				tmpl = append(
					tmpl,
					string(remaingPrt),
				)
				remaingPrt = []byte{}
				break
			}

			tmpl = append(
				tmpl,
				string(remaingPrt[:nextIndex]),
			)
		} else {
			nextIndex = indexAfterExpr(remaingPrt)
			if nextIndex == -1 {
				return nil, errors.New("Unclosed expression in attribute value")
			}

			exprs = append(
				exprs,
				string(remaingPrt[1:nextIndex-1]),
			)
		}

		remaingPrt = remaingPrt[nextIndex:]
	}

	if len(tmpl) == len(exprs) {
		tmpl = append(tmpl, "")
	}

	if len(tmpl) == 2 && len(tmpl[0]) == 0 && len(tmpl[1]) == 0 {
		return &exprAttr{
			name: string(namePrt),
			expr: string(exprs[0]),
		}, nil
	}

	return &tmplAttr{
		name:  string(namePrt),
		tmpl:  tmpl,
		exprs: exprs,
	}, nil
}

func stripInitWhiteSpace(data []byte) []byte {
	remaingData := data
	for len(remaingData) != 0 && unicode.IsSpace(rune(remaingData[0])) {
		remaingData = remaingData[1:]
	}
	return remaingData
}

func stripQuotes(data []byte) []byte {
	if len(data) <= 2 {
		return []byte{}
	}

	return data[1 : len(data)-1]
}

type staticAttr struct {
	name    string
	content string
}

func (attr *staticAttr) Name() string {
	return attr.name
}

func (attr *staticAttr) Content() string {
	return attr.content
}

func (attr *staticAttr) RewriteJs(_ js.VarRewriter) ([]byte, js.RewriteInfo) {
	data := []byte("'" + strings.ReplaceAll(attr.content, "'", `\'`) + "'")
	return data, js.NewEmptyRewriteInfo()
}

type exprAttr struct {
	name string
	expr string
}

func (attr *exprAttr) Name() string {
	return attr.name
}

func (attr *exprAttr) Content() string {
	return "{" + attr.expr + "}"
}

func (attr *exprAttr) RewriteJs(rw js.VarRewriter) ([]byte, js.RewriteInfo) {
	return rw.Rewrite([]byte(attr.expr))
}

type tmplAttr struct {
	name  string
	tmpl  []string
	exprs []string
}

func (attr *tmplAttr) Name() string {
	return attr.name
}

func (attr *tmplAttr) Content() string {
	c := attr.tmpl[0]
	for i, expr := range attr.exprs {
		c += "{" + expr + "}"
		c += attr.tmpl[i+1]
	}

	return c
}

func (attr *tmplAttr) RewriteJs(rw js.VarRewriter) ([]byte, js.RewriteInfo) {
	data := [][]byte{}
	data = append(data, []byte("`"))
	data = append(data, []byte(attr.tmpl[0]))

	allInfo := []js.RewriteInfo{}
	for i, expr := range attr.exprs {
		rwData, info := rw.Rewrite([]byte(expr))
		allInfo = append(allInfo, info)

		data = append(data, []byte("${"))
		data = append(data, rwData)
		data = append(data, []byte("}"))
		data = append(data, []byte(strings.ReplaceAll(attr.tmpl[i+1], "`", "\\`")))
	}

	data = append(data, []byte("`"))
	return bytes.Join(data, nil), js.MergeRewriteInfo(allInfo...)
}
