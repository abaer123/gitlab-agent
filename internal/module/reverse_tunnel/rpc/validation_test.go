package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/status"
)

type validatable interface {
	Validate() error
}

func TestValidation_Valid(t *testing.T) {
	tests := []struct {
		name  string
		valid validatable
	}{
		{
			name: "minimal",
			valid: &Error{
				Status: &status.Status{},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			assert.NoError(t, tc.valid.Validate()) // nolint: scopelint
		})
	}
}

func TestValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		errString string
		invalid   validatable
	}{
		{
			name:      "missing Error.Status",
			errString: "invalid Error.Status: value is required",
			invalid:   &Error{},
		},
		{
			name:      "empty AgentService.Name",
			errString: "invalid AgentService.Name: value length must be at least 1 runes",
			invalid:   &AgentService{},
		},
		{
			name:      "empty ServiceMethod.Name",
			errString: "invalid ServiceMethod.Name: value length must be at least 1 runes",
			invalid:   &ServiceMethod{},
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
