package agentcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:  "empty",
			valid: &AgentConfiguration{},
		},
		{
			name: "empty group id",
			valid: &ResourceFilterCF{
				ApiGroups: []string{""}, // empty string is ok
				Kinds:     []string{"ConfigMap"},
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
			name:      "empty ManifestProjects.Id",
			errString: "invalid ManifestProjectCF.Id: value length must be at least 1 runes",
			invalid: &ManifestProjectCF{
				Id: "", // empty id is not ok
			},
		},
		{
			name:      "empty list ResourceFilterCF.ApiGroups",
			errString: "invalid ResourceFilterCF.ApiGroups: value must contain at least 1 item(s)",
			invalid: &ResourceFilterCF{
				ApiGroups: []string{}, // empty list is not ok
			},
		},
		{
			name:      "empty list ResourceFilterCF.Kinds",
			errString: "invalid ResourceFilterCF.Kinds: value must contain at least 1 item(s)",
			invalid: &ResourceFilterCF{
				ApiGroups: []string{""},
				Kinds:     []string{}, // empty list is not ok
			},
		},
		{
			name:      "empty item string ResourceFilterCF.Kinds",
			errString: "invalid ResourceFilterCF.Kinds[0]: value length must be at least 1 runes",
			invalid: &ResourceFilterCF{
				ApiGroups: []string{""},
				Kinds:     []string{""}, // empty string is not ok
			},
		},
		{
			name:      "empty PathCF.Glob",
			errString: "invalid PathCF.Glob: value length must be at least 1 runes",
			invalid: &PathCF{
				Glob: "",
			},
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
