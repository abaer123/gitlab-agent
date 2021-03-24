package gitlab_test

import (
	"context"
	"errors"
	"net/http"
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
	_ gitlab.ResponseHandler = gitlab.ResponseHandlerStruct{}
)

func TestRequestOptions(t *testing.T) {
	ctx, correlationId := testhelpers.CtxWithCorrelation(t)
	c := mock_gitlab.SetupClient(t, "/ok", func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertRequestMethod(t, r, "CUSTOM_METHOD")
		testhelpers.AssertRequestAccept(t, r, "Bla")
		testhelpers.AssertAgentToken(t, r, testhelpers.AgentkToken)
		assert.Empty(t, r.Header.Values("Content-Type"))
		testhelpers.AssertCommonRequestParams(t, r, correlationId)
		testhelpers.AssertJWTSignature(t, r)
		assert.Equal(t, "val1", r.URL.Query().Get("key"))
		assert.Equal(t, "val2", r.Header.Get("h1"))
	})

	err := c.Do(ctx,
		gitlab.WithMethod("CUSTOM_METHOD"),
		gitlab.WithPath("/ok"),
		gitlab.WithQuery(url.Values{
			"key": []string{"val1"},
		}),
		gitlab.WithHeader(http.Header{
			"h1": []string{"val2"},
		}),
		gitlab.WithAgentToken(testhelpers.AgentkToken),
		gitlab.WithJWT(true),
		gitlab.WithResponseHandler(gitlab.ResponseHandlerStruct{
			AcceptHeader: "Bla",
			HandleFunc: func(resp *http.Response, err error) error {
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				// Do nothing
				return nil
			},
		}),
	)
	require.NoError(t, err)
}

func TestJsonResponseHandler_Forbidden(t *testing.T) {
	c := mock_gitlab.SetupClient(t, "/forbidden", func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequest(t, r)
		w.WriteHeader(http.StatusForbidden)
	})

	var resp interface{}

	err := c.Do(context.Background(),
		gitlab.WithPath("/forbidden"),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(&resp)),
	)
	require.Error(t, err)
	assert.True(t, gitlab.IsForbidden(err))
}

func TestJsonResponseHandler_Unauthorized(t *testing.T) {
	c := mock_gitlab.SetupClient(t, "/unauthorized", func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequest(t, r)
		w.WriteHeader(http.StatusUnauthorized)
	})

	var resp interface{}

	err := c.Do(context.Background(),
		gitlab.WithPath("/unauthorized"),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(&resp)),
	)
	require.Error(t, err)
	assert.True(t, gitlab.IsUnauthorized(err))
}

func TestJsonResponseHandler_HappyPath(t *testing.T) {
	c := mock_gitlab.SetupClient(t, "/ok", func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequest(t, r)
		testhelpers.RespondWithJSON(t, w, 42)
	})

	var resp interface{}

	err := c.Do(context.Background(),
		gitlab.WithPath("/ok"),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(&resp)),
	)
	require.NoError(t, err)
	assert.EqualValues(t, 42, resp)
}

func TestJsonResponseHandler_Cancellation(t *testing.T) {
	ctxClient, cancelClient := context.WithCancel(context.Background())
	defer cancelClient()
	cancelServer := make(chan struct{})
	c := mock_gitlab.SetupClient(t, "/cancel", func(w http.ResponseWriter, r *http.Request) {
		testhelpers.AssertGetJsonRequest(t, r)
		cancelClient() // unblock client
		<-cancelServer // wait for client to get the error and unblock server
	})

	var resp interface{}

	err := c.Do(ctxClient,
		gitlab.WithPath("/cancel"),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(&resp)),
	)
	close(cancelServer) // unblock server
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestNoContentResponseHandler_Forbidden(t *testing.T) {
	c := mock_gitlab.SetupClient(t, "/forbidden", func(w http.ResponseWriter, r *http.Request) {
		assertNoContentRequest(t, r)
		w.WriteHeader(http.StatusForbidden)
	})

	err := c.Do(context.Background(),
		gitlab.WithPath("/forbidden"),
		gitlab.WithResponseHandler(gitlab.NoContentResponseHandler()),
	)
	require.Error(t, err)
	assert.True(t, gitlab.IsForbidden(err))
}

func TestNoContentResponseHandler_Unauthorized(t *testing.T) {
	c := mock_gitlab.SetupClient(t, "/unauthorized", func(w http.ResponseWriter, r *http.Request) {
		assertNoContentRequest(t, r)
		w.WriteHeader(http.StatusUnauthorized)
	})

	err := c.Do(context.Background(),
		gitlab.WithPath("/unauthorized"),
		gitlab.WithResponseHandler(gitlab.NoContentResponseHandler()),
	)
	require.Error(t, err)
	assert.True(t, gitlab.IsUnauthorized(err))
}

func TestNoContentResponseHandler_HappyPath(t *testing.T) {
	c := mock_gitlab.SetupClient(t, "/ok", func(w http.ResponseWriter, r *http.Request) {
		assertNoContentRequest(t, r)
		testhelpers.RespondWithJSON(t, w, 42)
	})

	err := c.Do(context.Background(),
		gitlab.WithPath("/ok"),
		gitlab.WithResponseHandler(gitlab.NoContentResponseHandler()),
	)
	require.NoError(t, err)
}

func TestNoContentResponseHandler_Cancellation(t *testing.T) {
	ctxClient, cancelClient := context.WithCancel(context.Background())
	defer cancelClient()
	cancelServer := make(chan struct{})
	c := mock_gitlab.SetupClient(t, "/cancel", func(w http.ResponseWriter, r *http.Request) {
		assertNoContentRequest(t, r)
		cancelClient() // unblock client
		<-cancelServer // wait for client to get the error and unblock server
	})

	err := c.Do(ctxClient,
		gitlab.WithPath("/cancel"),
		gitlab.WithResponseHandler(gitlab.NoContentResponseHandler()),
	)
	close(cancelServer) // unblock server
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func assertNoContentRequest(t *testing.T, r *http.Request) {
	testhelpers.AssertRequestMethod(t, r, http.MethodGet)
	assert.Empty(t, r.Header.Values("Accept"))
}
