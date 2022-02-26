package html

import (
	"bytes"

	"github.com/progrium/sveltish/internal/js"
)

func indexStartExpr(data []byte) int {
	index := bytes.IndexByte(data, '{')
	if index == -1 {
		return -1
	}

	isEscaped := isCurrEscaped(data, index)
	if isEscaped {
		nextIndex := indexStartExpr(data[index+1:])
		if nextIndex == -1 {
			return -1
		}

		index += nextIndex + 1
	}

	return index
}

func indexAfterExpr(data []byte) int {
	return js.IndexAfterCurlyGroup(data)
}

func isCurrEscaped(data []byte, index int) bool {
	if index == 0 {
		return false
	}

	isEscaped := false
	lastIndex := index - 1
	for lastIndex > 0 && data[lastIndex] == '\\' {
		isEscaped = !isEscaped
		lastIndex -= 1
	}

	return isEscaped
}
