package components

import "github.com/catgoose/linkwell"

// ErrorSurface identifies which rendering surface an error should use.
type ErrorSurface string

const (
	SurfaceBanner     ErrorSurface = "banner"
	SurfaceInline     ErrorSurface = "inline"
	SurfaceInlineFull ErrorSurface = "inline-full"
	SurfaceFullPage   ErrorSurface = "full-page"
)

// ErrorSize is a semantic size hint for container-owning surfaces.
// Follows Tailwind conventions. Only meaningful for inline-full.
type ErrorSize string

// Standard size constants following Tailwind conventions.
const (
	SizeXS  ErrorSize = "xs"
	SizeSM  ErrorSize = "sm"
	SizeMD  ErrorSize = "md"
	SizeLG  ErrorSize = "lg"
	SizeXL  ErrorSize = "xl"
	Size2XL ErrorSize = "2xl"
	Size3XL ErrorSize = "3xl"
)

// ErrorPresentation is the unified caller-facing contract for rendering errors.
// The caller decides surface, size, and controls. Dothog decides how to render.
type ErrorPresentation struct {
	Surface   ErrorSurface
	Size      ErrorSize
	Title     string
	Detail    string
	Route     string
	RequestID string
	Theme     string
	OOBTarget string
	OOBSwap   string
	Controls  []linkwell.Control
	Status    int
	Closable  bool
}

// Normalize applies default values and surface-appropriate rules.
func (p *ErrorPresentation) Normalize() {
	if p.Surface == "" {
		p.Surface = SurfaceInline
	}
	if p.Size == "" {
		p.Size = SizeMD
	}
	if p.Status == 0 {
		p.Status = 500
	}
	if p.Title == "" {
		p.Title = "Error"
	}
	// Banner: force closable, set default OOB target.
	if p.Surface == SurfaceBanner {
		p.Closable = true
		if p.OOBTarget == "" {
			p.OOBTarget = linkwell.DefaultErrorStatusTarget
		}
	}
	// Full-page: dismiss makes no sense — strip dismiss-only controls.
	if p.Surface == SurfaceFullPage {
		p.Closable = false
		p.Controls = filterNonDismiss(p.Controls)
	}
}

func filterNonDismiss(controls []linkwell.Control) []linkwell.Control {
	out := make([]linkwell.Control, 0, len(controls))
	for _, c := range controls {
		if c.Kind != linkwell.ControlKindDismiss {
			out = append(out, c)
		}
	}
	return out
}

// toErrorContext converts an ErrorPresentation to a linkwell.ErrorContext
// for compatibility with existing template components.
func (p *ErrorPresentation) toErrorContext() linkwell.ErrorContext {
	return linkwell.ErrorContext{
		StatusCode: p.Status,
		Message:    p.Title,
		Route:      p.Route,
		RequestID:  p.RequestID,
		Controls:   p.Controls,
		Closable:   p.Closable,
		Theme:      p.Theme,
		OOBTarget:  p.OOBTarget,
		OOBSwap:    p.OOBSwap,
	}
}

// --- Ergonomic constructors ---

// NewBannerError creates a banner-surface error presentation.
func NewBannerError(status int, title string, controls ...linkwell.Control) ErrorPresentation {
	p := ErrorPresentation{
		Surface:  SurfaceBanner,
		Status:   status,
		Title:    title,
		Controls: controls,
	}
	p.Normalize()
	return p
}

// NewInlineError creates an inline-surface error presentation.
func NewInlineError(status int, title string, controls ...linkwell.Control) ErrorPresentation {
	return ErrorPresentation{
		Surface:  SurfaceInline,
		Status:   status,
		Title:    title,
		Controls: controls,
	}
}

// NewInlineFullError creates an inline-full error presentation with explicit size.
func NewInlineFullError(status int, title string, size ErrorSize, controls ...linkwell.Control) ErrorPresentation {
	p := ErrorPresentation{
		Surface:  SurfaceInlineFull,
		Status:   status,
		Title:    title,
		Size:     size,
		Controls: controls,
	}
	if p.Size == "" {
		p.Size = SizeMD
	}
	return p
}

// NewFullPageError creates a full-page error presentation.
func NewFullPageError(status int, title, detail, route, requestID, theme string, controls ...linkwell.Control) ErrorPresentation {
	p := ErrorPresentation{
		Surface:   SurfaceFullPage,
		Status:    status,
		Title:     title,
		Detail:    detail,
		Route:     route,
		RequestID: requestID,
		Theme:     theme,
		Controls:  controls,
	}
	p.Normalize()
	return p
}
