package info

import (
	"testing"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
)

func TestValidation_Invalid(t *testing.T) {
	tests := []testhelpers.InvalidTestcase{
		{
			Name:      "empty Service.Name",
			ErrString: "invalid Service.Name: value length must be at least 1 runes",
			Invalid:   &Service{},
		},
		{
			Name:      "empty Method.Name",
			ErrString: "invalid Method.Name: value length must be at least 1 runes",
			Invalid:   &Method{},
		},
	}
	testhelpers.AssertInvalid(t, tests)
}
