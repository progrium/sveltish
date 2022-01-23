package sveltish

import (
	"os"
	"path/filepath"
	"strings"
)

type Component struct {
	Name string
	JS   *Node
	HTML *Node
	CSS  *Node
}

type Node struct {
	Tag      string
	Text     string
	Children []*Node
	Parent   *Node
}

func walk(n *Node, fn func(n *Node) bool) bool {
	if n.Tag != "" {
		if !fn(n) {
			return false
		}
	}
	for _, child := range n.Children {
		if !walk(child, fn) {
			return false
		}
	}
	return true
}

func flatten(n *Node) []*Node {
	var nodes []*Node
	walk(n, func(n *Node) bool {
		nodes = append(nodes, n)
		return true
	})
	return nodes
}

func Build(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	name := strings.Replace(filepath.Base(path), filepath.Ext(path), "", 1)
	c, err := Parse(f, name)
	if err != nil {
		return nil, err
	}
	//spew.Dump(c)
	return GenerateJS(c)
}
