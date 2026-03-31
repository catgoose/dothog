package views

import (
	"fmt"

	"github.com/a-h/templ"
)

func debugOnChange(key string) templ.Attributes {
	return templ.Attributes{
		"onchange": fmt.Sprintf("window._dbg.toggle('%s', this.checked)", key),
	}
}
