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

	c, err := NewComponent(doc, name)
	if err != nil {
		return c, err
	}

	return c, nil
}
