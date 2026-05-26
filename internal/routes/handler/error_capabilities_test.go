package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	corecomponents "catgoose/dothog/web/components/core"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func newCapContext(t *testing.T, headers map[string]string) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestParseErrorCapabilities_Absent(t *testing.T) {
	caps := ParseErrorCapabilities(newCapContext(t, nil))
	assert.False(t, caps.HasAdvertised(), "no capability headers means no advertisement")
	assert.Empty(t, caps.Accept)
	assert.Empty(t, string(caps.Fallback))
}

func TestParseErrorCapabilities_AcceptListSplitsAndTrims(t *testing.T) {
	caps := ParseErrorCapabilities(newCapContext(t, map[string]string{
		HeaderErrorAcceptSurfaces: " inline , inline-full ,banner",
	}))
	assert.True(t, caps.HasAdvertised())
	assert.Equal(t, []corecomponents.ErrorSurface{
		corecomponents.SurfaceInline,
		corecomponents.SurfaceInlineFull,
		corecomponents.SurfaceBanner,
	}, caps.Accept)
}

func TestParseErrorCapabilities_FallbackTrims(t *testing.T) {
	caps := ParseErrorCapabilities(newCapContext(t, map[string]string{
		HeaderErrorFallbackSurface: "  banner ",
	}))
	assert.True(t, caps.HasAdvertised())
	assert.Equal(t, corecomponents.SurfaceBanner, caps.Fallback)
}

func TestNegotiateSurface_NoAdvertisementKeepsChosen(t *testing.T) {
	got := NegotiateSurface(corecomponents.SurfaceInline, ErrorCapabilities{})
	assert.Equal(t, corecomponents.SurfaceInline, got,
		"without advertisement the server's chosen surface passes through")
}

func TestNegotiateSurface_AcceptedSurfaceWins(t *testing.T) {
	caps := ErrorCapabilities{
		Accept:   []corecomponents.ErrorSurface{corecomponents.SurfaceInline, corecomponents.SurfaceBanner},
		Fallback: corecomponents.SurfaceBanner,
	}
	got := NegotiateSurface(corecomponents.SurfaceInline, caps)
	assert.Equal(t, corecomponents.SurfaceInline, got)
}

func TestNegotiateSurface_UnacceptedDowngradesToFallback(t *testing.T) {
	caps := ErrorCapabilities{
		Accept:   []corecomponents.ErrorSurface{corecomponents.SurfaceBanner},
		Fallback: corecomponents.SurfaceBanner,
	}
	got := NegotiateSurface(corecomponents.SurfaceInline, caps)
	assert.Equal(t, corecomponents.SurfaceBanner, got,
		"unaccepted chosen surface degrades to the client's declared fallback")
}

func TestNegotiateSurface_NoFallbackDefaultsToBanner(t *testing.T) {
	caps := ErrorCapabilities{
		Accept: []corecomponents.ErrorSurface{corecomponents.SurfaceInline},
	}
	got := NegotiateSurface(corecomponents.SurfacePage, caps)
	assert.Equal(t, corecomponents.SurfaceBanner, got,
		"missing fallback falls back to the safe banner default")
}
