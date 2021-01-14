package gitlab_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/testhelpers"
)

var (
	_ gitlab.ClientInterface = &gitlab.Client{}
)

func TestErrorCodes(t *testing.T) {
	ctxClient, cancelClient := context.WithCancel(context.Background())
	defer cancelClient()
	ctxServer, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()
	r := http.NewServeMux()
	r.HandleFunc("/forbidden", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	r.HandleFunc("/unauthorized", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	r.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		cancelClient()     // unblock client
		<-ctxServer.Done() // wait for client to get the error and unblock server
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	c := gitlab.NewClient(u, []byte(testhelpers.AuthSecretKey), mock_gitlab.ClientOptionsForTest()...)

	err = c.DoJSON(context.Background(), http.MethodGet, "/forbidden", nil, testhelpers.AgentkToken, nil, nil)
	require.Error(t, err)
	assert.True(t, gitlab.IsForbidden(err))

	err = c.DoJSON(context.Background(), http.MethodGet, "/unauthorized", nil, testhelpers.AgentkToken, nil, nil)
	require.Error(t, err)
	assert.True(t, gitlab.IsUnauthorized(err))

	err = c.DoJSON(ctxClient, http.MethodGet, "/cancel", nil, testhelpers.AgentkToken, nil, nil)
	cancelServer() // unblock server
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}
