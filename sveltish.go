package sveltish

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/progrium/sveltish/internal/html"
)

type Component struct {
	Name string
	JS   *html.Doc
	HTML *html.Doc
	CSS  *html.Doc
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

	return GenerateJS(c)
}
