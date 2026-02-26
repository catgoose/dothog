package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// toJSONString
// ---------------------------------------------------------------------------

func TestToJSONString_EmptyMap(t *testing.T) {
	result := toJSONString(map[string]string{})
	require.Equal(t, "{}", result)
}

func TestToJSONString_SingleKey(t *testing.T) {
	result := toJSONString(map[string]string{"key": "value"})
	require.Equal(t, `{"key":"value"}`, result)
}

func TestToJSONString_MultipleKeys(t *testing.T) {
	data := map[string]string{
		"name":  "required",
		"email": "invalid",
	}
	result := toJSONString(data)

	require.True(t, strings.HasPrefix(result, "{"), "should start with {")
	require.True(t, strings.HasSuffix(result, "}"), "should end with }")
	require.Contains(t, result, `"name":"required"`)
	require.Contains(t, result, `"email":"invalid"`)
}

// ---------------------------------------------------------------------------
// splitPascalCase
// ---------------------------------------------------------------------------

func TestSplitPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"two words", "FirstName", "first name"},
		{"three words", "UserPrincipalName", "user principal name"},
		{"consecutive caps", "ID", "id"},
		{"already lowercase", "email", "email"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPascalCase(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// formatValidationError
// ---------------------------------------------------------------------------

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		tag      string
		param    string
		expected string
	}{
		{"required", "FirstName", "required", "", "first name is required"},
		{"required_if", "FirstName", "required_if", "", "first name is required"},
		{"oneof", "Status", "oneof", "active inactive", "status must be one of: active inactive"},
		{"min", "Age", "min", "18", "age must be at least 18"},
		{"max", "Name", "max", "100", "name cannot exceed 100"},
		{"default case", "Email", "email", "", "Invalid email: email"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValidationError(tt.field, tt.tag, tt.param)
			require.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// getFieldName
// ---------------------------------------------------------------------------

func TestGetFieldName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"two words", "FirstName", "first-name"},
		{"single word", "Email", "email"},
		{"three words", "UserPrincipalName", "user-principal-name"},
		{"consecutive caps", "ID", "id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// FormValidationMiddleware — integration tests
// ---------------------------------------------------------------------------

type testFormModel struct {
	Name  string `form:"name" validate:"required"`
	Email string `form:"email" validate:"required"`
	Age   int    `form:"age" validate:"min=1,max=150"`
}

func TestFormValidationMiddleware_GETPassesThrough(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	model := &testFormModel{}
	mw := FormValidationMiddleware(model)
	called := false
	handler := mw(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)

	require.NoError(t, err)
	require.True(t, called, "next handler should be called for GET")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestFormValidationMiddleware_POSTValidData(t *testing.T) {
	e := echo.New()
	formData := "name=John&email=john@test.com&age=25"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	model := &testFormModel{}
	mw := FormValidationMiddleware(model)
	called := false
	handler := mw(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)

	require.NoError(t, err)
	require.True(t, called, "next handler should be called for valid POST")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("HX-Trigger"), "clearValidation")

	stored := c.Get("validatedModel")
	require.NotNil(t, stored, "validated model should be stored in context")
}

func TestFormValidationMiddleware_POSTMissingRequiredFields(t *testing.T) {
	e := echo.New()
	formData := "age=0"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	model := &testFormModel{}
	mw := FormValidationMiddleware(model)
	called := false
	handler := mw(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)

	require.NoError(t, err)
	require.False(t, called, "next handler should NOT be called for invalid POST")
	require.Equal(t, http.StatusOK, rec.Code)

	trigger := rec.Header().Get("HX-Trigger")
	require.Contains(t, trigger, "formValidationErrors")
	require.Equal(t, "none", rec.Header().Get("HX-Reswap"))
}

func TestFormValidationMiddleware_PUTValidData(t *testing.T) {
	e := echo.New()
	formData := "name=Jane&email=jane@test.com&age=30"
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	model := &testFormModel{}
	mw := FormValidationMiddleware(model)
	called := false
	handler := mw(func(c echo.Context) error {
		called = true
		return c.String(http.StatusOK, "ok")
	})

	err := handler(c)

	require.NoError(t, err)
	require.True(t, called, "next handler should be called for valid PUT")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("HX-Trigger"), "clearValidation")

	stored := c.Get("validatedModel")
	require.NotNil(t, stored, "validated model should be stored in context")
}
