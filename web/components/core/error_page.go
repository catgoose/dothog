package components

import "catgoose/dothog/internal/routes/hypermedia"

func errorPageTheme(ec hypermedia.ErrorContext) string {
	if ec.Theme != "" {
		return ec.Theme
	}
	return "light"
}
