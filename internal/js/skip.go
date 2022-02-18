package js

type skipper interface {
	isOpen() bool
	next(b byte)
}

type groupSkipper struct {
	count      int
	open       byte
	close      byte
	canComment bool
	inner      skipper
}

func newGroupSkipper(open, close byte) *groupSkipper {
	return &groupSkipper{
		count:      1,
		open:       open,
		close:      close,
		canComment: false,
		inner:      nil,
	}
}

func newCurlyGroupSkipper() *groupSkipper {
	return newGroupSkipper(byte(curlyOpen[0]), byte(curlyClose[0]))
}

func newParenGroupSkipper() *groupSkipper {
	return newGroupSkipper(byte(parenOpen[0]), byte(parenClose[0]))
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
	case gt.open:
		gt.count += 1
	case gt.close:
		gt.count -= 1
	case byte(lineCommentOpen[1]):
		if !gt.canComment {
			return
		}
		gt.inner = newCommentSkipper(lineCommentOpen, newLine)
	case byte(blockCommentOpen[1]):
		if !gt.canComment {
			return
		}
		gt.inner = newCommentSkipper(blockCommentOpen, blockCommentClose)
	case byte(singleQuote[0]), byte(doubleQuote[0]), byte(tmplQuote[0]), byte(regexQuote[0]):
		gt.inner = newQuoteSkipper(c)
	}
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
			qt.inner = newGroupSkipper(byte(tmplQuoteExprOpen[1]), byte(tmplQuoteExprClose[0]))
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

type commentSkipper struct {
	open  bool
	close string
	last  byte
}

func newCommentSkipper(open, close string) *commentSkipper {
	return &commentSkipper{
		open:  true,
		close: close,
		last:  byte(open[len(open)-1]),
	}
}

func newBlockCommentSkipper() *commentSkipper {
	return newCommentSkipper(blockCommentOpen, blockCommentClose)
}

func newLineCommentSkipper() *commentSkipper {
	return newCommentSkipper(lineCommentOpen, newLine)
}

func (ct *commentSkipper) isOpen() bool {
	return ct.open
}

func (ct *commentSkipper) next(c byte) {
	defer func() {
		ct.last = c
	}()

	switch len(ct.close) {
	case 1:
		if c == byte(ct.close[0]) {
			ct.open = false
		}
	case 2:
		if ct.last == byte(ct.close[0]) && c == byte(ct.close[1]) {
			ct.open = false
		}
	}
}
