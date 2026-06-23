package components

import (
	"strconv"

	"github.com/a-h/templ"
	"github.com/catgoose/linkwell"
)

// hxAttrsFromControl converts HxRequest fields to templ.Attributes with "hx-" prefix.
// Also injects hx-confirm, hx-push-url, and hx-swap from their dedicated Control fields.
func hxAttrsFromControl(ctrl linkwell.Control) templ.Attributes {
	req := ctrl.HxRequest
	attrs := make(templ.Attributes, 5)
	if req.URL != "" {
		attrs["hx-"+string(req.Method)] = req.URL
	}
	if req.Target != "" {
		attrs["hx-target"] = req.Target
	}
	if req.Include != "" {
		attrs["hx-include"] = req.Include
	}
	if req.Vals != "" {
		attrs["hx-vals"] = req.Vals
	}
	if ctrl.Confirm != "" {
		attrs["hx-confirm"] = ctrl.Confirm
	}
	if ctrl.PushURL != "" {
		attrs["hx-push-url"] = ctrl.PushURL
	}
	if ctrl.Swap != "" {
		attrs["hx-swap"] = string(ctrl.Swap)
	}
	if ctrl.ErrorTarget != "" {
		attrs["hx-target-error"] = ctrl.ErrorTarget
	}
	return attrs
}

// DetailsDropdownAttrs is the _hyperscript wiring for a standalone <details>
// dropdown: spread it onto the <details> to close the menu on an outside click
// and, when focusSelector is non-empty, focus the first matching descendant
// each time it opens. The lookup runs inside the element (me.querySelector), so
// duplicate desktop/mobile copies never steal focus from each other.
//
// Pass "" for close-on-outside only. The navbar shell already closes its own
// nested <details>; reach for this on dropdowns outside that shell.
func DetailsDropdownAttrs(focusSelector string) templ.Attributes {
	script := `on click from window
  if me.open and event.target is not within me
    set me.open to false
  end`
	if focusSelector != "" {
		script += `
on toggle
  if me.open
    set focusTarget to me.querySelector(` + strconv.Quote(focusSelector) + `)
    if focusTarget
      call focusTarget.focus()
    end
  end`
	}
	return templ.Attributes{"_": script}
}
