package components

import (
	"testing"

	"catgoose/go-htmx-demo/internals/routes/hypermedia"

	"github.com/stretchr/testify/require"
)

func TestHxAttrsFromControl_BasicAttrs(t *testing.T) {
	ctrl := hypermedia.Control{
		HxRequest: hypermedia.HxGet("/items", "#list"),
		Swap:      hypermedia.SwapInnerHTML,
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/items", attrs["hx-get"])
	require.Equal(t, "#list", attrs["hx-target"])
	require.Equal(t, "innerHTML", attrs["hx-swap"])
}

func TestHxAttrsFromControl_Confirm(t *testing.T) {
	ctrl := hypermedia.Control{
		Confirm:   "Are you sure?",
		HxRequest: hypermedia.HxDelete("/item/1", ""),
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "Are you sure?", attrs["hx-confirm"])
	require.Equal(t, "/item/1", attrs["hx-delete"])
}

func TestHxAttrsFromControl_PushURL(t *testing.T) {
	ctrl := hypermedia.Control{
		PushURL:   "/dashboard",
		HxRequest: hypermedia.HxGet("/dashboard", ""),
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/dashboard", attrs["hx-push-url"])
	require.Equal(t, "/dashboard", attrs["hx-get"])
}

func TestHxAttrsFromControl_SwapField(t *testing.T) {
	ctrl := hypermedia.Control{
		Swap:      hypermedia.SwapOuterHTML,
		HxRequest: hypermedia.HxPost("/submit", ""),
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "outerHTML", attrs["hx-swap"])
	require.Equal(t, "/submit", attrs["hx-post"])
}

func TestHxAttrsFromControl_AllFieldsSet(t *testing.T) {
	ctrl := hypermedia.Control{
		HxRequest: hypermedia.HxPut("/update", ""),
		Confirm:   "Confirm?",
		PushURL:   "/updated",
		Swap:      hypermedia.SwapNone,
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/update", attrs["hx-put"])
	require.Equal(t, "Confirm?", attrs["hx-confirm"])
	require.Equal(t, "/updated", attrs["hx-push-url"])
	require.Equal(t, "none", attrs["hx-swap"])
}

func TestHxAttrsFromControl_EmptyControl(t *testing.T) {
	ctrl := hypermedia.Control{}
	attrs := hxAttrsFromControl(ctrl)
	require.NotNil(t, attrs)
	_, hasConfirm := attrs["hx-confirm"]
	_, hasPush := attrs["hx-push-url"]
	_, hasSwap := attrs["hx-swap"]
	require.False(t, hasConfirm)
	require.False(t, hasPush)
	require.False(t, hasSwap)
}

func TestHxAttrsFromControl_IncludeField(t *testing.T) {
	ctrl := hypermedia.Control{
		HxRequest: hypermedia.HxRequestConfig{
			Method:  hypermedia.HxMethodPut,
			URL:     "/save",
			Target:  "#tc",
			Include: "closest tr",
		},
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/save", attrs["hx-put"])
	require.Equal(t, "#tc", attrs["hx-target"])
	require.Equal(t, "closest tr", attrs["hx-include"])
}

func TestHxAttrsFromControl_ZeroHxRequest(t *testing.T) {
	ctrl := hypermedia.Control{
		Confirm: "Sure?",
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "Sure?", attrs["hx-confirm"])
}
