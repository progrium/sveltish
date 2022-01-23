package js

import (
	"fmt"
	"testing"
)

func TestGen(t *testing.T) {
	js := &Source{}
	js.Stmt(`import {
  SvelteComponent,
  append,
  detach,
  element,
  init,
  insert,
  noop,
  safe_not_equal
} from`, js.Str("./runtime"))
	js.Line("")
	js.Func("create_fragment", []string{"ctx"}, func(js *Source) {
		js.Stmt("let main")
		js.Line("")
		js.Stmt("return", func(js *Source) {
			js.Stmt("c()", func(js *Source) {
				js.Stmt(`main = element("main")`)
			}, ",")
			js.Line("p: noop,")
			js.Line("i: noop,")
		})
	})
	fmt.Println(js.String())
}
