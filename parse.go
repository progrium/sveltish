package sveltish

import (
	"io"

	"github.com/progrium/sveltish/internal/html"
)

func Parse(name string, src io.Reader) (*Component, error) {
	doc, err := html.Parse(src)
	if err != nil {
		return nil, err
	}

	c, err := NewComponent(name, doc)
	if err != nil {
		return c, err
	}

	return c, nil
}
