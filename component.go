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

	var err error
	html.Walk(doc, func(n html.Node, ps []html.NodeContainer) bool {
		if err != nil {
			return false
		}

		switch len(ps) {
		case 0:
			return true
		case 1:
			err = c.appendRoot(n)
			return false
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

	var err error
	html.Walk(n, func(n html.Node, _ []html.NodeContainer) bool {
		if err != nil {
			return false
		}

		//TODO, split into useful parts for generating svelet JS

		if ln, ok := n.(*html.LeafElNode); ok && (ln.Tag == "script" || ln.Tag == "style") {
			err = errors.New("Connot add <script /> or <style /> element other than as a root element")
			return false
		}
		return true
	})

	if err != nil {
		return err
	}
	return nil
}
