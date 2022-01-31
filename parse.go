package sveltish

import (
	"io"

	"github.com/progrium/sveltish/internal/html"
)

func Parse(src io.Reader, name string) (*Component, error) {
	doc, err := html.Parse(src)
	if err != nil {
		return nil, err
	}

	c := &Component{
		Name: name,
		JS:   &html.Doc{},
		HTML: &html.Doc{},
		CSS:  &html.Doc{},
	}
	html.Walk(doc, func(n html.Node, ps []html.NodeContainer) bool {
		if len(ps) == 0 {
			return true
		}

		if lf, ok := n.(*html.LeafElNode); ok {
			switch lf.Tag {
			case "script":
				c.JS.AppendChild(lf)
			case "style":
				c.CSS.AppendChild(lf)
			}
			return false
		}

		c.HTML.AppendChild(n)
		return false
	})

	return c, nil
}
