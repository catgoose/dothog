// setup:feature:demo
package middleware

import (
	"net/http"
	"testing"

	hypermedia "github.com/catgoose/linkwell"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetLinks(t *testing.T) {
	t.Helper()
	t.Cleanup(hypermedia.ResetForTesting)
	hypermedia.ResetForTesting()
}

func TestLinkRelationsMiddleware_LinkHeaderPresent(t *testing.T) {
	resetLinks(t)

	hypermedia.Link("/items", "related", "/items/new", "New Item")

	c, rec := newTestContext(http.MethodGet, "/items")

	mw := LinkRelationsMiddleware()
	handler := mw(func(c echo.Context) error {
		return nil
	})
	err := handler(c)

	require.NoError(t, err)
	linkHeader := rec.Header().Get("Link")
	assert.Contains(t, linkHeader, "/items/new")
	assert.Contains(t, linkHeader, `rel="related"`)
}

func TestLinkRelationsMiddleware_ContextHasLinks(t *testing.T) {
	resetLinks(t)

	hypermedia.Link("/items", "related", "/items/new", "New Item")

	c, _ := newTestContext(http.MethodGet, "/items")

	var captured []hypermedia.LinkRelation
	mw := LinkRelationsMiddleware()
	handler := mw(func(c echo.Context) error {
		captured = GetLinkRelations(c)
		return nil
	})
	err := handler(c)

	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Len(t, captured, 1)
	assert.Equal(t, "/items/new", captured[0].Href)
}

func TestLinkRelationsMiddleware_NoLinksForPath(t *testing.T) {
	resetLinks(t)

	c, rec := newTestContext(http.MethodGet, "/unknown")

	var captured []hypermedia.LinkRelation
	mw := LinkRelationsMiddleware()
	handler := mw(func(c echo.Context) error {
		captured = GetLinkRelations(c)
		return nil
	})
	err := handler(c)

	require.NoError(t, err)
	assert.Empty(t, rec.Header().Get("Link"), "no Link header when no links registered")
	assert.Nil(t, captured, "context should have no link_relations")
}

func TestGetLinkRelations_ReturnsNilWhenNotSet(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/")
	links := GetLinkRelations(c)
	assert.Nil(t, links)
}

func TestGetLinkRelations_ReturnsLinksFromContext(t *testing.T) {
	c, _ := newTestContext(http.MethodGet, "/")
	expected := []hypermedia.LinkRelation{
		{Rel: "related", Href: "/b", Title: "B"},
	}
	c.Set("link_relations", expected)

	links := GetLinkRelations(c)
	require.NotNil(t, links)
	assert.Equal(t, expected, links)
}
