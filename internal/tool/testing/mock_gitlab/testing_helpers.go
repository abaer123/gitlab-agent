package mock_gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func AssertGitalyRepository(t *testing.T, gitalyRepository gitlab.GitalyRepository, apiGitalyRepository gitalypb.Repository) {
	assert.Equal(t, gitalyRepository.StorageName, apiGitalyRepository.StorageName)
	assert.Equal(t, gitalyRepository.RelativePath, apiGitalyRepository.RelativePath)
	assert.Equal(t, gitalyRepository.GlRepository, apiGitalyRepository.GlRepository)
	assert.Equal(t, gitalyRepository.GlProjectPath, apiGitalyRepository.GlProjectPath)
}

func AssertGitalyInfo(t *testing.T, gitalyInfo gitlab.GitalyInfo, apiGitalyInfo api.GitalyInfo) {
	assert.Equal(t, gitalyInfo.Address, apiGitalyInfo.Address)
	assert.Equal(t, gitalyInfo.Token, apiGitalyInfo.Token)
	assert.Equal(t, gitalyInfo.Features, apiGitalyInfo.Features)
}

func ClientOptionsForTest() []gitlab.ClientOption {
	return []gitlab.ClientOption{
		gitlab.WithUserAgent(testhelpers.KasUserAgent),
		gitlab.WithCorrelationClientName(testhelpers.KasCorrelationClientName),
	}
}
