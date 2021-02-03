package tracker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/reverse_tunnel/info"
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
			valid: &TunnelInfo{
				AgentDescriptor: &info.AgentDescriptor{},
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
			name:      "missing TunnelInfo.AgentDescriptor",
			errString: "invalid TunnelInfo.AgentDescriptor: value is required",
			invalid:   &TunnelInfo{},
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
