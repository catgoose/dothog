// Package htmx provides HX-* header constants and helpers for echo handlers.
// This package has no imports from this project.
package htmx

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
)

// HX-* response header names.
const (
	HeaderTrigger            = "HX-Trigger"
	HeaderTriggerAfterSettle = "HX-Trigger-After-Settle"
	HeaderRedirect           = "HX-Redirect"
	HeaderPushURL            = "HX-Push-Url"
	HeaderReplaceURL         = "HX-Replace-Url"
	HeaderReswap             = "HX-Reswap"
	HeaderRetarget           = "HX-Retarget"
	HeaderReselect           = "HX-Reselect"
	HeaderRefresh            = "HX-Refresh"
	HeaderLocation           = "HX-Location"
)

// SwapStrategy represents an HTMX swap strategy value.
type SwapStrategy string

// HTMX swap strategy constants.
const (
	SwapInnerHTML   SwapStrategy = "innerHTML"
	SwapOuterHTML   SwapStrategy = "outerHTML"
	SwapBeforeBegin SwapStrategy = "beforebegin"
	SwapAfterBegin  SwapStrategy = "afterbegin"
	SwapBeforeEnd   SwapStrategy = "beforeend"
	SwapAfterEnd    SwapStrategy = "afterend"
	SwapDelete      SwapStrategy = "delete"
	SwapNone        SwapStrategy = "none"
)

// IsHTMX reports whether the request was made by HTMX.
func IsHTMX(c echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}

// Trigger sets the HX-Trigger response header to the given raw JSON string.
func Trigger(c echo.Context, jsonStr string) {
	c.Response().Header().Set(HeaderTrigger, jsonStr)
}

// TriggerEvent marshals event+payload into JSON and sets HX-Trigger.
func TriggerEvent(c echo.Context, event string, payload any) error {
	data := map[string]any{event: payload}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.Response().Header().Set(HeaderTrigger, string(b))
	return nil
}

// TriggerAfterSettle sets the HX-Trigger-After-Settle response header.
func TriggerAfterSettle(c echo.Context, jsonStr string) {
	c.Response().Header().Set(HeaderTriggerAfterSettle, jsonStr)
}

// Redirect sets HX-Redirect, causing HTMX to perform a client-side redirect.
func Redirect(c echo.Context, url string) {
	c.Response().Header().Set(HeaderRedirect, url)
}

// PushURL sets HX-Push-Url to push a new URL into the browser history.
func PushURL(c echo.Context, url string) {
	c.Response().Header().Set(HeaderPushURL, url)
}

// ReplaceURL sets HX-Replace-Url to replace the current URL in history.
func ReplaceURL(c echo.Context, url string) {
	c.Response().Header().Set(HeaderReplaceURL, url)
}

// Reswap sets HX-Reswap to override the swap strategy for the current request.
func Reswap(c echo.Context, strategy SwapStrategy) {
	c.Response().Header().Set(HeaderReswap, string(strategy))
}

// ReswapNone sets HX-Reswap to "none" so HTMX does not perform any swap.
func ReswapNone(c echo.Context) {
	Reswap(c, SwapNone)
}

// Retarget sets HX-Retarget to override the target CSS selector.
func Retarget(c echo.Context, cssSelector string) {
	c.Response().Header().Set(HeaderRetarget, cssSelector)
}

// Refresh sets HX-Refresh to trigger a full page refresh.
func Refresh(c echo.Context) {
	c.Response().Header().Set(HeaderRefresh, "true")
}
