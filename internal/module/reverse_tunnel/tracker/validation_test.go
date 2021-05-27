package tracker

import (
	"testing"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/reverse_tunnel/info"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
)

func TestValidation_Valid(t *testing.T) {
	tests := []testhelpers.ValidTestcase{
		{
			Name: "minimal",
			Valid: &TunnelInfo{
				AgentDescriptor: &info.AgentDescriptor{},
			},
		},
		{
			Name: "grpc",
			Valid: &TunnelInfo{
				AgentDescriptor: &info.AgentDescriptor{},
				KasUrl:          "grpc://1.1.1.1:10",
			},
		},
		{
			Name: "grpcs",
			Valid: &TunnelInfo{
				AgentDescriptor: &info.AgentDescriptor{},
				KasUrl:          "grpcs://1.1.1.1:10",
			},
		},
	}
	testhelpers.AssertValid(t, tests)
}

func TestValidation_Invalid(t *testing.T) {
	tests := []testhelpers.InvalidTestcase{
		{
			Name:      "missing TunnelInfo.AgentDescriptor",
			ErrString: "invalid TunnelInfo.AgentDescriptor: value is required",
			Invalid:   &TunnelInfo{},
		},
		{
			Name:      "invalid TunnelInfo.KasUrl",
			ErrString: `invalid TunnelInfo.KasUrl: value does not match regex pattern "(?:^$|^grpcs?://)"`,
			Invalid: &TunnelInfo{
				AgentDescriptor: &info.AgentDescriptor{},
				KasUrl:          "tcp://1.1.1.1:12",
			},
		},
	}
	testhelpers.AssertInvalid(t, tests)
}
