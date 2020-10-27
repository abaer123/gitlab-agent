package agentcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidation_Valid(t *testing.T) {
	tests := []struct {
		name string
		cfg  *AgentConfiguration
	}{
		{
			name: "valid - empty",
			cfg:  &AgentConfiguration{},
		},
		{
			name: "valid - empty group id",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceExclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{""}, // empty string is ok
									Kinds:     []string{"ConfigMap"},
								},
							},
							ResourceInclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{""}, // empty string is ok
									Kinds:     []string{"ConfigMap"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			assert.NoError(t, tc.cfg.Validate()) // nolint: scopelint
		})
	}
}

func TestValidation_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		errString string
		cfg       *AgentConfiguration
	}{
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.Id: value length must be at least 1 runes",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "", // empty id is not ok
						},
					},
				},
			},
		},
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.ResourceExclusions[0]: embedded message failed validation | caused by: invalid ResourceFilterCF.ApiGroups: value must contain at least 1 item(s)",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceExclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{}, // empty list is not ok
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.ResourceExclusions[0]: embedded message failed validation | caused by: invalid ResourceFilterCF.Kinds: value must contain at least 1 item(s)",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceExclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{""},
									Kinds:     []string{}, // empty list is not ok
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.ResourceExclusions[0]: embedded message failed validation | caused by: invalid ResourceFilterCF.Kinds[0]: value length must be at least 1 runes",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceExclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{""},
									Kinds:     []string{""}, // empty string is not ok
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.ResourceInclusions[0]: embedded message failed validation | caused by: invalid ResourceFilterCF.ApiGroups: value must contain at least 1 item(s)",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceInclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{}, // empty list is not ok
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.ResourceInclusions[0]: embedded message failed validation | caused by: invalid ResourceFilterCF.Kinds: value must contain at least 1 item(s)",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceInclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{""},
									Kinds:     []string{}, // empty list is not ok
								},
							},
						},
					},
				},
			},
		},
		{
			name:      "invalid - empty Gitops.ManifestProjects.Id",
			errString: "invalid AgentConfiguration.Gitops: embedded message failed validation | caused by: invalid GitopsCF.ManifestProjects[0]: embedded message failed validation | caused by: invalid ManifestProjectCF.ResourceInclusions[0]: embedded message failed validation | caused by: invalid ResourceFilterCF.Kinds[0]: value length must be at least 1 runes",
			cfg: &AgentConfiguration{
				Gitops: &GitopsCF{
					ManifestProjects: []*ManifestProjectCF{
						{
							Id: "123",
							ResourceInclusions: []*ResourceFilterCF{
								{
									ApiGroups: []string{""},
									Kinds:     []string{""}, // empty string is not ok
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			err := tc.cfg.Validate() // nolint: scopelint
			require.Error(t, err)
			assert.EqualError(t, err, tc.errString) // nolint: scopelint
		})
	}
}
