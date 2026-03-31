package components

import hypermedia "github.com/catgoose/linkwell"

func errorPageTheme(ec hypermedia.ErrorContext) string {
	if ec.Theme != "" {
		return ec.Theme
	}
	return "light"
}
