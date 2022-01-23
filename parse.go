package sveltish

import (
	"io"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/html"
)

func Parse(src io.Reader, name string) (*Component, error) {
	l := html.NewLexer(parse.NewInput(src))
	loop := true
	root := &Node{}
	curr := root
	var last *Node
	for loop {
		tt, data := l.Next()
		switch tt {
		case html.ErrorToken:
			err := l.Err()
			if err != io.EOF {
				return nil, err
			}
			loop = false
			break
		case html.StartTagToken:
			tag := string(data[1:])
			node := &Node{Tag: tag, Parent: curr}
			curr.Children = append(curr.Children, node)
			last = curr
			curr = node
			break
		case html.EndTagToken:
			curr = last
			break
		case html.TextToken:
			curr.Text = strings.Trim(string(data), "\n\t ")
			break
		}
	}
	c := &Component{
		Name: name,
		JS:   &Node{},
		HTML: &Node{},
		CSS:  &Node{},
	}
	walk(root, func(n *Node) bool {
		switch n.Tag {
		case "script":
			c.JS.Children = append(c.JS.Children, n)
			break
		case "style":
			c.CSS.Children = append(c.CSS.Children, n)
			break
		default:
			c.HTML.Children = append(c.HTML.Children, n)
		}
		return true
	})
	return c, nil
}
