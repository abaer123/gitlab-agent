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
			Name:      "empty PathCF.Glob",
			ErrString: "invalid PathCF.Glob: value length must be at least 1 runes",
			Invalid: &PathCF{
				Glob: "",
			},
		},
	}
	testhelpers.AssertInvalid(t, tests)
}
