package components

import (
	"testing"

	"github.com/catgoose/linkwell"

	"github.com/stretchr/testify/require"
)

func TestHxAttrsFromControl_BasicAttrs(t *testing.T) {
	ctrl := linkwell.Control{
		HxRequest: linkwell.HxGet("/items", "#list"),
		Swap:      linkwell.SwapInnerHTML,
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/items", attrs["hx-get"])
	require.Equal(t, "#list", attrs["hx-target"])
	require.Equal(t, "innerHTML", attrs["hx-swap"])
}

func TestHxAttrsFromControl_Confirm(t *testing.T) {
	ctrl := linkwell.Control{
		Confirm:   "Are you sure?",
		HxRequest: linkwell.HxDelete("/item/1", ""),
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "Are you sure?", attrs["hx-confirm"])
	require.Equal(t, "/item/1", attrs["hx-delete"])
}

func TestHxAttrsFromControl_PushURL(t *testing.T) {
	ctrl := linkwell.Control{
		PushURL:   "/dashboard",
		HxRequest: linkwell.HxGet("/dashboard", ""),
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/dashboard", attrs["hx-push-url"])
	require.Equal(t, "/dashboard", attrs["hx-get"])
}

func TestHxAttrsFromControl_SwapField(t *testing.T) {
	ctrl := linkwell.Control{
		Swap:      linkwell.SwapOuterHTML,
		HxRequest: linkwell.HxPost("/submit", ""),
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "outerHTML", attrs["hx-swap"])
	require.Equal(t, "/submit", attrs["hx-post"])
}

func TestHxAttrsFromControl_AllFieldsSet(t *testing.T) {
	ctrl := linkwell.Control{
		HxRequest: linkwell.HxPut("/update", ""),
		Confirm:   "Confirm?",
		PushURL:   "/updated",
		Swap:      linkwell.SwapNone,
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "/update", attrs["hx-put"])
	require.Equal(t, "Confirm?", attrs["hx-confirm"])
	require.Equal(t, "/updated", attrs["hx-push-url"])
	require.Equal(t, "none", attrs["hx-swap"])
}

func TestHxAttrsFromControl_EmptyControl(t *testing.T) {
	ctrl := linkwell.Control{}
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
	ctrl := linkwell.Control{
		HxRequest: linkwell.HxRequestConfig{
			Method:  linkwell.HxMethodPut,
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
	ctrl := linkwell.Control{
		Confirm: "Sure?",
	}
	attrs := hxAttrsFromControl(ctrl)
	require.Equal(t, "Sure?", attrs["hx-confirm"])
}

func TestDetailsDropdownAttrs_CloseOnly(t *testing.T) {
	script, ok := DetailsDropdownAttrs("")["_"].(string)
	require.True(t, ok)
	require.Contains(t, script, "on click from window")
	require.Contains(t, script, "event.target is not within me")
	require.Contains(t, script, "set me.open to false")
	require.NotContains(t, script, "on toggle")
}

func TestDetailsDropdownAttrs_WithFocus(t *testing.T) {
	script, ok := DetailsDropdownAttrs("[data-search]")["_"].(string)
	require.True(t, ok)
	require.Contains(t, script, "on click from window")
	require.Contains(t, script, "on toggle")
	require.Contains(t, script, `me.querySelector("[data-search]")`)
	require.Contains(t, script, "call focusTarget.focus()")
}

func TestDetailsDropdownAttrs_FocusSelectorWithQuotes(t *testing.T) {
	script, ok := DetailsDropdownAttrs(`[data-label="Bob's menu"]`)["_"].(string)
	require.True(t, ok)
	require.Contains(t, script, `me.querySelector("[data-label=\"Bob's menu\"]")`)
}
