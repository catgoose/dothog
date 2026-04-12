package components

import "github.com/catgoose/linkwell"

func shellTheme(theme string) string {
	if theme != "" {
		return theme
	}
	return "light"
}

func errorDetail(ec linkwell.ErrorContext) string {
	if ec.Err != nil {
		return ec.Err.Error()
	}
	return ""
}
