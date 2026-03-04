package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type mockAction struct {
	calledIDs []int
	failOn    map[int]bool
}

func (m *mockAction) fn(_ context.Context, id int) error {
	m.calledIDs = append(m.calledIDs, id)
	if m.failOn[id] {
		return fmt.Errorf("mock error for id %d", id)
	}
	return nil
}

func bulkContext(ids ...string) echo.Context {
	e := echo.New()
	form := "ids=" + strings.Join(ids, "&ids=")
	req := httptest.NewRequest(http.MethodPost, "/tables/bulk/items", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestDoBulkAction_ValidIDs(t *testing.T) {
	b := &bulkRoutes{}
	m := &mockAction{}
	c := bulkContext("1", "2", "3")

	failed := b.doBulkAction(c, m.fn)
	assert.Empty(t, failed)
	assert.Equal(t, []int{1, 2, 3}, m.calledIDs)
}

func TestDoBulkAction_InvalidIDs(t *testing.T) {
	b := &bulkRoutes{}
	m := &mockAction{}
	c := bulkContext("abc", "0", "-1")

	failed := b.doBulkAction(c, m.fn)
	assert.Empty(t, failed)
	assert.Empty(t, m.calledIDs)
}

func TestDoBulkAction_MixedIDs(t *testing.T) {
	b := &bulkRoutes{}
	m := &mockAction{failOn: map[int]bool{2: true}}
	c := bulkContext("1", "abc", "2", "3")

	failed := b.doBulkAction(c, m.fn)
	assert.Equal(t, []int{2}, failed)
	assert.Equal(t, []int{1, 2, 3}, m.calledIDs)
}

func TestDoBulkAction_EmptyForm(t *testing.T) {
	b := &bulkRoutes{}
	m := &mockAction{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/tables/bulk/items", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	failed := b.doBulkAction(c, m.fn)
	assert.Empty(t, failed)
	assert.Empty(t, m.calledIDs)
}
