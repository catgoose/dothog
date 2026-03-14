// setup:feature:graph

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToNullString_NonEmpty(t *testing.T) {
	ns := ToNullString("hello")
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)
}

func TestToNullString_Empty(t *testing.T) {
	ns := ToNullString("")
	assert.False(t, ns.Valid)
	assert.Empty(t, ns.String)
}
