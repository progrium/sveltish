package js

type skipper interface {
	isOpen() bool
	next(byte)
	group() ([]byte, []byte)
}

// IndexAfterCurlyGroup finds the index after
func IndexAfterCurlyGroup(data []byte) int {
	if data[0] != byte(curlyOpen[0]) {
		panic("Trying to skip curly group in byte slice that doesn't start with {")
	}

	skpr := newCurlyGroupSkipper()
	for i, b := range data[1:] {
		skpr.next(b)
		if skpr.isOpen() {
			continue
		}

		return i + 2
	}
	return -1
}

type groupSkipper struct {
	count      int
	open       []byte
	close      []byte
	canComment bool
	inner      skipper
}

func newGroupSkipper(open, close []byte) *groupSkipper {
	return &groupSkipper{
		count:      1,
		open:       open,
		close:      close,
		canComment: false,
		inner:      nil,
	}
}

func newCurlyGroupSkipper() *groupSkipper {
	return newGroupSkipper([]byte(curlyOpen), []byte(curlyClose))
}

func newParenGroupSkipper() *groupSkipper {
	return newGroupSkipper([]byte(parenOpen), []byte(parenClose))
}

func newTmplGroupSkipper() *groupSkipper {
	return newGroupSkipper([]byte(tmplQuoteExprOpen), []byte(tmplQuoteExprClose))
}

func (gt *groupSkipper) isOpen() bool {
	return gt.count != 0
}

func (gt *groupSkipper) next(c byte) {
	if gt.inner != nil {
		gt.inner.next(c)
		if !gt.inner.isOpen() {
			gt.inner = nil
		}
		return
	}

	if c == '/' && !gt.canComment {
		gt.canComment = true
		return
	}
	defer func() {
		gt.canComment = false
	}()

	switch c {
	case gt.open[len(gt.open)-1]:
		gt.count += 1
	case gt.close[0]:
		gt.count -= 1
	case byte(lineCommentOpen[1]):
		if !gt.canComment {
			return
		}
		gt.inner = newLineCommentSkipper()
	case byte(blockCommentOpen[1]):
		if !gt.canComment {
			return
		}
		gt.inner = newBlockCommentSkipper()
	case byte(singleQuote[0]):
		gt.inner = newSingleQuoteSkipper()
	case byte(doubleQuote[0]):
		gt.inner = newDoubleQuoteSkipper()
	case byte(tmplQuote[0]):
		gt.inner = newTmplQuoteSkipper()
	case byte(regexQuote[0]):
		gt.inner = newRegexQuoteSkipper()
	}
}

func (gt *groupSkipper) group() ([]byte, []byte) {
	if gt.inner != nil && gt.inner.isOpen() {
		return gt.inner.group()
	}

	return gt.open, gt.close
}

type quoteSkipper struct {
	quote    byte
	open     bool
	escaped  bool
	canGroup bool
	inner    skipper
}

func newQuoteSkipper(quote byte) *quoteSkipper {
	return &quoteSkipper{
		quote:    quote,
		open:     true,
		escaped:  false,
		canGroup: false,
		inner:    nil,
	}
}

func newSingleQuoteSkipper() *quoteSkipper {
	return newQuoteSkipper(byte(singleQuote[0]))
}

func newDoubleQuoteSkipper() *quoteSkipper {
	return newQuoteSkipper(byte(doubleQuote[0]))
}

func newTmplQuoteSkipper() *quoteSkipper {
	return newQuoteSkipper(byte(tmplQuote[0]))
}

func newRegexQuoteSkipper() *quoteSkipper {
	return newQuoteSkipper(byte(regexQuote[0]))
}

func (qt *quoteSkipper) isOpen() bool {
	return qt.open
}

func (qt *quoteSkipper) next(c byte) {
	if qt.inner != nil {
		qt.inner.next(c)
		if !qt.inner.isOpen() {
			qt.inner = nil
		}
		return
	}

	if qt.quote != byte(regexQuote[0]) && c == byte(quoteEscape[0]) {
		qt.escaped = !qt.escaped
		qt.canGroup = false
		return
	}
	if qt.escaped {
		qt.escaped = false
		return
	}

	if qt.quote == byte(tmplQuote[0]) {
		if qt.canGroup && c == byte(tmplQuoteExprOpen[1]) {
			qt.canGroup = false
			qt.inner = newTmplGroupSkipper()
			return
		}

		if c == byte(tmplQuoteExprOpen[0]) {
			qt.canGroup = true
			return
		}
		qt.canGroup = false
	}

	if c == qt.quote {
		qt.open = false
		return
	}
}

func (qt *quoteSkipper) group() ([]byte, []byte) {
	if qt.inner != nil && qt.inner.isOpen() {
		return qt.inner.group()
	}

	openAndClose := []byte{qt.quote}
	return openAndClose, openAndClose
}

type commentSkipper struct {
	open   []byte
	close  []byte
	closed bool
	last   byte
}

func newCommentSkipper(open, close []byte) *commentSkipper {
	return &commentSkipper{
		open:   open,
		close:  close,
		closed: false,
		last:   open[len(open)-1],
	}
}

func newBlockCommentSkipper() *commentSkipper {
	return newCommentSkipper([]byte(blockCommentOpen), []byte(blockCommentClose))
}

func newLineCommentSkipper() *commentSkipper {
	return newCommentSkipper([]byte(lineCommentOpen), []byte(newLine))
}

func (ct *commentSkipper) isOpen() bool {
	return !ct.closed
}

func (ct *commentSkipper) next(c byte) {
	defer func() {
		ct.last = c
	}()

	switch len(ct.close) {
	case 1:
		if c == ct.close[0] {
			ct.closed = true
		}
	case 2:
		if ct.last == ct.close[0] && c == ct.close[1] {
			ct.closed = true
		}
	default:
		panic("Comment close cannot be more than 2 chars long")
	}
}

func (ct *commentSkipper) group() ([]byte, []byte) {
	return ct.open, ct.close
}
