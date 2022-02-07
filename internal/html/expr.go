package html

import (
	"bytes"
)

type group string

const (
	curlyGroup    group = "{}"
	singleQuote   group = "'"
	doubleQuote   group = "\""
	rawQuote      group = "`"
	allGroupChars       = string(curlyGroup + singleQuote + doubleQuote + rawQuote)
)

func indexStartExpr(data []byte) int {
	index := bytes.IndexByte(data, curlyGroup[0])
	if index == -1 {
		return -1
	}

	isEscaped := isCharEscaped(data, index)
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
	gt := newExprTracker(data, curlyGroup)
	gt.findClose()

	return gt.index
}

type exprTracker struct {
	data  []byte
	stack []group
	index int
}

func newExprTracker(data []byte, initGroup group) *exprTracker {
	if data[0] != initGroup[0] {
		panic("Trying to find the end of a not started group, split using result of btyes.IndexByte(data, initGroup[0]) first")
	}

	return &exprTracker{
		data,
		[]group{initGroup},
		0,
	}
}

func (et *exprTracker) isOpen() bool {
	return len(et.stack) != 0
}

func (et *exprTracker) push(charGroup group) {
	et.stack = append(et.stack, charGroup)
}

func (et *exprTracker) pop() {
	et.stack = et.stack[:len(et.stack)-1]
}

func (et *exprTracker) peek() group {
	return et.stack[len(et.stack)-1]
}

func (et *exprTracker) peekGroup() (group, bool) {
	currGroup := et.peek()
	if currGroup != curlyGroup {
		return "", false
	}

	return currGroup, true
}

func (et *exprTracker) peekQuote() (group, bool) {
	currGroup := et.peek()
	if currGroup != singleQuote && currGroup != doubleQuote && currGroup != rawQuote {
		return "", false
	}

	return currGroup, true
}

func (et *exprTracker) findClose() {
	for et.isOpen() {
		et.next()
		if et.index == -1 {
			return
		}
	}

	et.index += 1
}

func (et *exprTracker) next() {
	et.index += 1
	nextIndex := bytes.IndexAny(et.data[et.index:], allGroupChars)
	if nextIndex == -1 {
		et.index = -1
		return
	}

	et.index += nextIndex
	currChar := et.data[et.index]

	if isEscaped := isCharEscaped(et.data, et.index); isEscaped {
		return
	}

	if quoteChars, inQuote := et.peekQuote(); inQuote {
		if currChar == quoteChars[0] {
			et.pop()
		}
		return
	}

	for _, quoteType := range []group{singleQuote, doubleQuote, rawQuote} {
		if quoteType[0] == currChar {
			et.push(quoteType)
			return
		}
	}

	switch currChar {
	case curlyGroup[0]:
		et.push(curlyGroup)
		return
	case curlyGroup[1]:
		if _, inGroup := et.peekGroup(); !inGroup {
			et.index = -1
			return
		}

		et.pop()
		return
	}

	panic("bytes.IndexAny found char not in allGroupChars")
}

func isCharEscaped(data []byte, index int) bool {
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
