package agentcfg

import (
	"testing"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
)

func TestValidation_Valid(t *testing.T) {
	tests := []testhelpers.ValidTestcase{
		{
			Name:  "empty",
			Valid: &AgentConfiguration{},
		},
		{
			Name: "empty group id",
			Valid: &ResourceFilterCF{
				ApiGroups: []string{""}, // empty string is ok
				Kinds:     []string{"ConfigMap"},
			},
		},
	}
	testhelpers.AssertValid(t, tests)
}

func TestValidation_Invalid(t *testing.T) {
	tests := []testhelpers.InvalidTestcase{
		{
			Name:      "empty ManifestProjects.Id",
			ErrString: "invalid ManifestProjectCF.Id: value length must be at least 1 runes",
			Invalid: &ManifestProjectCF{
				Id: "", // empty id is not ok
			},
		},
		{
			Name:      "empty list ResourceFilterCF.ApiGroups",
			ErrString: "invalid ResourceFilterCF.ApiGroups: value must contain at least 1 item(s)",
			Invalid: &ResourceFilterCF{
				ApiGroups: []string{}, // empty list is not ok
			},
		},
		{
			Name:      "empty list ResourceFilterCF.Kinds",
			ErrString: "invalid ResourceFilterCF.Kinds: value must contain at least 1 item(s)",
			Invalid: &ResourceFilterCF{
				ApiGroups: []string{""},
				Kinds:     []string{}, // empty list is not ok
			},
		},
		{
			Name:      "empty item string ResourceFilterCF.Kinds",
			ErrString: "invalid ResourceFilterCF.Kinds[0]: value length must be at least 1 runes",
			Invalid: &ResourceFilterCF{
				ApiGroups: []string{""},
				Kinds:     []string{""}, // empty string is not ok
			},
		},
		{
			Name:      "empty PathCF.Glob",
			ErrString: "invalid PathCF.Glob: value length must be at least 1 runes",
			Invalid: &PathCF{
				Glob: "",
			},
		},
	}
	testhelpers.AssertInvalid(t, tests)
}
