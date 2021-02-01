package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type validatable interface {
	Validate() error
}

func TestValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		errString string
		invalid   validatable
	}{
		{
			name:      "empty Service.Name",
			errString: "invalid Service.Name: value length must be at least 1 runes",
			invalid:   &Service{},
		},
		{
			name:      "empty Method.Name",
			errString: "invalid Method.Name: value length must be at least 1 runes",
			invalid:   &Method{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			err := tc.invalid.Validate() // nolint: scopelint
			require.Error(t, err)
			assert.EqualError(t, err, tc.errString) // nolint: scopelint
		})
	}
}
