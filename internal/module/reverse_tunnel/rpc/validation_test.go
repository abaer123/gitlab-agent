package rpc

import (
	"testing"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"google.golang.org/genproto/googleapis/rpc/status"
)

func TestValidation_Valid(t *testing.T) {
	tests := []testhelpers.ValidTestcase{
		{
			Name: "minimal",
			Valid: &Error{
				Status: &status.Status{},
			},
		},
	}
	testhelpers.AssertValid(t, tests)
}

func TestValidation_Invalid(t *testing.T) {
	tests := []testhelpers.InvalidTestcase{
		{
			Name:      "missing Error.Status",
			ErrString: "invalid Error.Status: value is required",
			Invalid:   &Error{},
		},
	}
	testhelpers.AssertInvalid(t, tests)
}
