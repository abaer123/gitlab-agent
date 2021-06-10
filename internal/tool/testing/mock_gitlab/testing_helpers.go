package mock_gitlab

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"go.uber.org/zap/zaptest"
)

func AssertGitalyRepository(t *testing.T, gitalyRepository gitlab.GitalyRepository, apiGitalyRepository *gitalypb.Repository) {
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

func SetupClient(t *testing.T, pattern string, handler func(http.ResponseWriter, *http.Request)) *gitlab.Client {
	r := http.NewServeMux()
	r.HandleFunc(pattern, handler)
	s := httptest.NewServer(r)
	t.Cleanup(s.Close)

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	return gitlab.NewClient(u, []byte(testhelpers.AuthSecretKey),
		gitlab.WithUserAgent(testhelpers.KasUserAgent),
		gitlab.WithCorrelationClientName(testhelpers.KasCorrelationClientName),
		gitlab.WithLogger(zaptest.NewLogger(t)),
	)
}
