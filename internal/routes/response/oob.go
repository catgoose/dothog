package response

import (
	"context"
	"io"

	hypermedia "github.com/catgoose/linkwell"
	corecomponents "catgoose/dothog/web/components/core"

	"github.com/a-h/templ"
)

// OOBPlacement describes an out-of-band component swap.
type OOBPlacement struct {
	Component templ.Component
	Target    string
	Swap      string
}

// ToComponent returns the component to be rendered as OOB content.
func (p OOBPlacement) ToComponent() templ.Component {
	return p.Component
}

// ErrorOOB creates an OOB placement that renders the error context into #error-status.
// It sets ec.OOBTarget (defaulting to "#error-status") before rendering.
func ErrorOOB(ec hypermedia.ErrorContext) OOBPlacement {
	if ec.OOBTarget == "" {
		ec.OOBTarget = hypermedia.DefaultErrorStatusTarget
	}
	if ec.OOBSwap == "" {
		ec.OOBSwap = "innerHTML"
	}
	return OOBPlacement{
		Target:    ec.OOBTarget,
		Swap:      ec.OOBSwap,
		Component: corecomponents.ErrorStatusFromContext(ec),
	}
}

// ClearErrorOOB returns an OOB placement that clears the #error-status element.
func ClearErrorOOB() OOBPlacement {
	cmp := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<div id="error-status" hx-swap-oob="innerHTML"></div>`)
		return err
	})
	return OOBPlacement{
		Target:    hypermedia.DefaultErrorStatusTarget,
		Swap:      "innerHTML",
		Component: cmp,
	}
}
