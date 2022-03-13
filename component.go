package sveltish

import (
	"errors"
	"fmt"
	"strings"

	"github.com/progrium/sveltish/internal/html"
	"github.com/progrium/sveltish/internal/js"
)

type Component struct {
	Name string
	JS   *js.Script
	CSS  []*html.LeafElNode
	HTML []*NodeVar
}

func NewComponent(name string, doc *html.Doc) (*Component, error) {
	c := &Component{
		Name: name,
		JS:   nil,
		CSS:  []*html.LeafElNode{},
		HTML: []*NodeVar{},
	}
	nt := newNameTracker()
	err := html.Walk(doc, func(n html.Node, ps html.Parents) (bool, error) {
		switch ps.Depth() {
		case 0:
			return true, nil
		case 1:
			if ln, ok := n.(*html.LeafElNode); ok {
				switch ln.Tag() {
				case "script":
					if c.JS != nil {
						return false, errors.New("More than one <script /> element found")
					}

					script, err := js.Parse(strings.NewReader(ln.Content()))
					if err != nil {
						return false, err
					}
					c.JS = script
					return false, nil
				case "style":
					c.CSS = append(c.CSS, ln)
					return false, nil
				}
			}
		default:
			if ln, ok := n.(*html.LeafElNode); ok {
				if tag := ln.Tag(); tag == "script" || tag == "style" {
					return false, errors.New("Connot add <script /> or <style /> element other than as a root element")
				}
			}
		}

		if err := nt.addTrackingFor(n); err != nil {
			panic(err)
		}
		return true, nil
	})
	if err != nil {
		return c, err
	}

	html.Walk(doc, func(n html.Node, ps html.Parents) (bool, error) {
		switch ps.Depth() {
		case 0:
			return true, nil
		case 1:
			if ln, ok := n.(*html.LeafElNode); ok {
				if tag := ln.Tag(); tag == "script" || tag == "style" {
					return false, nil
				}
			}
		}

		name, err := nt.createName(n)
		if err != nil {
			panic(err)
		}

		if parentNode, exists := ps.Parent(); exists {
			if parentName, exists := nt.nameOf(parentNode); exists {
				c.HTML = append(c.HTML, NewNodeVarWithParent(name, parentName, n))
				return true, nil
			}
		}

		c.HTML = append(c.HTML, NewNodeVar(name, n))
		return true, nil
	})

	if c.JS == nil {
		c.JS = &js.Script{}
	}

	return c, nil
}

type NodeVar struct {
	name       string
	hasParent  bool
	parentName string
	node       html.Node
}

func NewNodeVar(name string, node html.Node) *NodeVar {
	return &NodeVar{
		name:      name,
		hasParent: false,
		node:      node,
	}
}

func NewNodeVarWithParent(name string, parentName string, node html.Node) *NodeVar {
	return &NodeVar{
		name:       name,
		hasParent:  true,
		parentName: parentName,
		node:       node,
	}
}

type nameTracker struct {
	pfxs  map[string]prefixData
	names map[html.NodeId]string
}

type prefixData struct {
	used  int
	total int
}

func newNameTracker() *nameTracker {
	return &nameTracker{
		pfxs:  map[string]prefixData{},
		names: map[html.NodeId]string{},
	}
}

func (nt *nameTracker) addTrackingFor(n html.Node) error {
	if len(nt.names) != 0 {
		return errors.New("Cannot add tracking once names have started be created")
	}

	pfx, exists := prefixFor(n)
	if !exists {
		return errors.New("Cannot add tracking for non text nodes that don't have a tag")
	}

	if data, exists := nt.pfxs[pfx]; exists {
		data.total += 1
		nt.pfxs[pfx] = data
		return nil
	}

	nt.pfxs[pfx] = prefixData{0, 1}
	return nil
}

func (nt *nameTracker) createName(n html.Node) (string, error) {
	pfx, exists := prefixFor(n)
	if !exists {
		return "", errors.New("Cannot create name for non text nodes that don't have a tag")
	}

	data, exist := nt.pfxs[pfx]
	if !exist || data.used > data.total {
		return "", errors.New("Cannot create name for nodes that were not tracked")
	}

	var name string
	if data.total == 1 {
		name = pfx
	} else {
		name = fmt.Sprintf("%s%d", pfx, data.used)
	}

	data.used += 1
	nt.pfxs[pfx] = data
	nt.names[n.Id()] = name

	return name, nil
}

func (nt *nameTracker) nameOf(n html.Node) (string, bool) {
	name, exists := nt.names[n.Id()]
	return name, exists
}

func prefixFor(n html.Node) (string, bool) {
	switch node := n.(type) {
	case html.Element:
		return node.Tag(), true
	case *html.TxtNode, *html.ExprNode:
		return "t", true
	}

	return "", false
}
