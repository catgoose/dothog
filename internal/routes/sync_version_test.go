package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResourceURL_Tasks(t *testing.T) {
	table, id, ok := parseResourceURL("/demo/repository/tasks/42")
	assert.True(t, ok)
	assert.Equal(t, "Tasks", table)
	assert.Equal(t, 42, id)
}

func TestParseResourceURL_Items(t *testing.T) {
	table, id, ok := parseResourceURL("/demo/items/7")
	assert.True(t, ok)
	assert.Equal(t, "Items", table)
	assert.Equal(t, 7, id)
}

func TestParseResourceURL_People(t *testing.T) {
	table, id, ok := parseResourceURL("/demo/people/3")
	assert.True(t, ok)
	assert.Equal(t, "People", table)
	assert.Equal(t, 3, id)
}

func TestParseResourceURL_NoMatch(t *testing.T) {
	_, _, ok := parseResourceURL("/health")
	assert.False(t, ok)
}

func TestParseResourceURL_NoID(t *testing.T) {
	_, _, ok := parseResourceURL("/demo/repository/tasks")
	assert.False(t, ok)
}

func TestParseResourceURL_CreateURL(t *testing.T) {
	// Create URLs don't have an ID — they shouldn't match
	_, _, ok := parseResourceURL("/demo/repository/tasks")
	assert.False(t, ok)
}

func TestIsValidTableName(t *testing.T) {
	assert.True(t, isValidTableName("Tasks"))
	assert.True(t, isValidTableName("Items"))
	assert.True(t, isValidTableName("my_table"))
	assert.False(t, isValidTableName("Tasks; DROP TABLE"))
	assert.False(t, isValidTableName(""))
	assert.False(t, isValidTableName("123"))
}
