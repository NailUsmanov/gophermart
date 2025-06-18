package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidLuhn(t *testing.T) {
	validator := &LuhnValidation{}
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"correct", "79927398713", true},
		{"Invalid Luhn", "79927398710", false},
		{"Non-numeric", "abc123", false},
		{"Empty string", "", false},
		{"Single digit", "5", false},
		{"All zeros", "0000000000000000", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.IsValidLuhn(tt.input)
			assert.Equal(t, tt.expect, result)
		})

	}
}
