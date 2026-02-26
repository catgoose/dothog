package components

import (
	"catgoose/go-htmx-demo/internals/routes/hypermedia"
	"github.com/a-h/templ"
)

// hxAttrsFromControl converts HxRequest fields to templ.Attributes with "hx-" prefix.
// Also injects hx-confirm, hx-push-url, and hx-swap from their dedicated Control fields.
func hxAttrsFromControl(ctrl hypermedia.Control) templ.Attributes {
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
