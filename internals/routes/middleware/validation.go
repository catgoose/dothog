package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"catgoose/go-htmx-demo/internals/logger"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
)

// toJSONString converts a map of field names to error messages into a JSON string
func toJSONString(data map[string]string) string {
	var builder strings.Builder
	builder.WriteString("{")
	first := true
	for key, value := range data {
		if !first {
			builder.WriteString(",")
		}
		builder.WriteString(fmt.Sprintf(`"%s":"%s"`, key, value))
		first = false
	}
	builder.WriteString("}")
	return builder.String()
}

// splitPascalCase converts PascalCase field names to user-friendly format
func splitPascalCase(input string) string {
	// Regular expression to find capital letters that start a new word
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	result := re.ReplaceAllString(input, `$1 $2`)
	// Convert to lowercase
	return strings.ToLower(result)
}

// formatValidationError formats a validation error message
func formatValidationError(field string, tag string, param string) string {
	fieldName := splitPascalCase(field)

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", fieldName)
	case "required_if":
		return fmt.Sprintf("%s is required", fieldName)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fieldName, param)
	case "min":
		return fmt.Sprintf("%s must be at least %s", fieldName, param)
	case "max":
		return fmt.Sprintf("%s cannot exceed %s", fieldName, param)
	default:
		return fmt.Sprintf("Invalid %s: %s", fieldName, tag)
	}
}

// formatValidationErrorWithCustomMessage formats a validation error message with custom message support
func formatValidationErrorWithCustomMessage(model any, field string, tag string, param string) string {
	// Use reflection to check for custom validation message
	val := reflect.ValueOf(model)
	// Dereference pointer to get the underlying struct
	if val.IsValid() && val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	typ := val.Type()
	if field, ok := typ.FieldByName(field); ok {
		if customMsg := field.Tag.Get("validate_msg"); customMsg != "" {
			return customMsg
		}
	}

	// Fall back to default formatting
	return formatValidationError(field, tag, param)
}

// getFieldName maps struct field names to form field names
func getFieldName(structField string) string {
	// Convert PascalCase to kebab-case for form field names
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	result := re.ReplaceAllString(structField, `$1-$2`)
	return strings.ToLower(result)
}

// FormValidationMiddleware is a generic middleware that binds and validates any struct
func FormValidationMiddleware(modelType any) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only handle POST and PUT requests
			if c.Request().Method != "POST" && c.Request().Method != "PUT" {
				return next(c)
			}

			// Create a new instance of the provided model type
			modelValue := modelType

			// Bind request data to the model
			if err := c.Bind(modelValue); err != nil {
				return BadRequest(c, "Failed to bind form data")
			}

			// Validate the form data using manual validation
			validate := validator.New()
			if err := validate.Struct(modelValue); err != nil {
				var validationErrs validator.ValidationErrors
				if errors.As(err, &validationErrs) {
					errorMap := make(map[string]string)
					for _, e := range validationErrs {
						fieldName := getFieldName(e.Field())
						errorMap[fieldName] = formatValidationErrorWithCustomMessage(modelValue, e.Field(), e.Tag(), e.Param())
					}
					fields := make([]string, 0, len(errorMap))
					for k := range errorMap {
						fields = append(fields, k)
					}
					logger.Warn("Validation failed",
						"route", c.Request().URL.Path,
						"fields", fields,
					)
					errorJSON := `{"formValidationErrors": ` + toJSONString(errorMap) + `}`
					c.Response().Header().Set("HX-Trigger", errorJSON)
					// Tell HTMX not to swap any content, just process the HX-Trigger
					c.Response().Header().Set("HX-Reswap", "none")
					return c.String(http.StatusOK, "")
				}
				// If it's not a validation error, set a generic error header
				logger.Warn("Non-ValidationErrors validation failure",
					"route", c.Request().URL.Path,
					"error", err.Error(),
				)
				c.Response().Header().Set("HX-Trigger", `{"formValidationErrors": {"general": "Form validation failed"}}`)
				c.Response().Header().Set("HX-Reswap", "none")
				return c.String(http.StatusOK, "")
			}
			// Clear previous validation errors if validation passed
			c.Response().Header().Set("HX-Trigger", `{"clearValidation": true}`)

			// Store validated data in context
			c.Set("validatedModel", modelValue)
			return next(c)
		}
	}
}
