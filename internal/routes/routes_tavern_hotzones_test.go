// setup:feature:demo

package routes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHexColor(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"#22c55e", true},
		{"#AABBCC", true},
		{"#000000", true},
		{"#ffffff", true},
		{"#1e293b", true},
		{"", false},
		{"22c55e", false},       // missing #
		{"#abc", false},         // too short
		{"#1234567", false},     // too long
		{"#gggggg", false},      // invalid hex chars
		{"rgb(0,0,0)", false},   // not hex
		{"red", false},          // named color
		{"#12345g", false},      // invalid char
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isHexColor(tt.input))
		})
	}
}
