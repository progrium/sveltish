package sveltish

import (
	"errors"

	"github.com/progrium/sveltish/internal/html"
)

type Component struct {
	Name string
	JS   []*html.LeafElNode
	CSS  []*html.LeafElNode
	HTML *html.Doc //TODO, split into useful parts for generating svelet JS
}

func NewComponent(name string, doc *html.Doc) (*Component, error) {
	c := &Component{
		Name: name,
		JS:   []*html.LeafElNode{},
		CSS:  []*html.LeafElNode{},
		HTML: &html.Doc{},
	}

	err := html.Walk(doc, func(n html.Node, ps []html.NodeContainer) (bool, error) {
		switch len(ps) {
		case 0:
			return true, nil
		case 1:
			if err := c.appendRoot(n); err != nil {
				return false, err
			}
			return false, nil
		}
		panic("html.Walk(...) in NewComponent(...) should never go more than depth 1")
	})
	if err != nil {
		return c, err
	}

	return c, nil
}

func (c *Component) appendRoot(n html.Node) error {
	if ln, ok := n.(*html.LeafElNode); ok {
		switch ln.Tag {
		case "script":
			c.JS = append(c.JS, ln)
			return nil
		case "style":
			c.CSS = append(c.CSS, ln)
			return nil
		}
	}
	c.HTML.AppendChild(n)

	err := html.Walk(n, func(n html.Node, _ []html.NodeContainer) (bool, error) {
		//TODO, split into useful parts for generating svelet JS
		if ln, ok := n.(*html.LeafElNode); ok && (ln.Tag == "script" || ln.Tag == "style") {
			return false, errors.New("Connot add <script /> or <style /> element other than as a root element")
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}
