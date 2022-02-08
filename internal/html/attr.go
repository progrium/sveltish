package html

import (
	"bytes"
	"errors"
	"strings"
)

// An attr represents an attribute on an html element.
type Attr interface {
	JsContenter
	Name() string

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
	for len(remaingData) != 0 && isWhiteSpace(remaingData[0]) {
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

func (attr *staticAttr) JsContent() string {
	return "'" + strings.ReplaceAll(attr.content, "'", `\'`) + "'"
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

func (attr *exprAttr) JsContent() string {
	return attr.expr
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

func (attr *tmplAttr) JsContent() string {
	c := attr.tmpl[0]
	for i, expr := range attr.exprs {
		c += "${" + expr + "}"
		c += strings.ReplaceAll(attr.tmpl[i+1], "`", "\\`")
	}

	return "`" + c + "`"
}
