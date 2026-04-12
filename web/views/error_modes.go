// setup:feature:demo
package views

import (
	"errors"
	"net/http"

	corecomponents "catgoose/dothog/web/components/core"

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

func errorModesInlineFullRetryEC(size string) linkwell.ErrorContext {
	target := "#errors-modes-inline-full-" + size
	retryURL := "/patterns/errors/modes/inline-full/" + size
	return linkwell.ErrorContext{
		StatusCode: http.StatusTooManyRequests,
		Message:    "Too Many Requests",
		Err:        errors.New("rate limit exceeded for this panel"),
		Route:      retryURL,
		RequestID:  "req-demo-429-inline",
		Closable:   true,
		Controls: []linkwell.Control{
			linkwell.RetryButton(linkwell.LabelRetry, linkwell.HxMethodGet, retryURL, target),
			linkwell.DismissButton(linkwell.LabelDismiss),
		},
	}
}

// ContractDemoPresentations returns sample ErrorPresentations for the contract demo.
func ContractDemoPresentations() []corecomponents.ErrorPresentation {
	return []corecomponents.ErrorPresentation{
		corecomponents.NewBannerError(500, "Background job failed",
			linkwell.DismissButton(linkwell.LabelDismiss),
			linkwell.ReportIssueButton(linkwell.LabelReportIssue, "req-contract-500"),
		),
		corecomponents.NewInlineError(422, "Validation failed",
			linkwell.DismissButton(linkwell.LabelDismiss),
		),
		corecomponents.NewInlineFullError(429, "Rate limited", corecomponents.SizeMD,
			linkwell.DismissButton(linkwell.LabelDismiss),
		),
	}
}

// ContractFullPagePresentation returns a full-page ErrorPresentation for the contract demo.
func ContractFullPagePresentation(theme string) corecomponents.ErrorPresentation {
	return corecomponents.NewFullPageError(
		503, "Service Unavailable",
		"A downstream dependency is not responding. This page was rendered through the unified error contract.",
		"/patterns/errors/modes/contract/full-page",
		"req-contract-503",
		theme,
		linkwell.GoHomeButton(linkwell.LabelGoHome, "/", ""),
		linkwell.ReportIssueButton(linkwell.LabelReportIssue, "req-contract-503"),
	)
}
