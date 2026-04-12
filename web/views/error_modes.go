// setup:feature:demo
package views

import (
	"errors"
	"net/http"

	"github.com/catgoose/linkwell"
)

func errorModesInlineEC() linkwell.ErrorContext {
	return linkwell.ErrorContext{
		StatusCode: http.StatusUnprocessableEntity,
		Message:    "Validation failed",
		Err:        errors.New("the submitted data could not be processed"),
		Route:      "/patterns/errors/modes/inline",
		Closable:   true,
		Controls: []linkwell.Control{
			linkwell.DismissButton(linkwell.LabelDismiss),
		},
	}
}

func errorModesInlineFullEC() linkwell.ErrorContext {
	return linkwell.ErrorContext{
		StatusCode: http.StatusTooManyRequests,
		Message:    "Too Many Requests",
		Err:        errors.New("rate limit exceeded for this panel"),
		Route:      "/patterns/errors/modes/inline-full",
		RequestID:  "req-demo-429-inline",
		Closable:   true,
		Controls: []linkwell.Control{
			linkwell.DismissButton(linkwell.LabelDismiss),
		},
	}
}
