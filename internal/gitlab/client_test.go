package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/correlation"
)

const (
	kasUserAgent                            = "kas/v0.1-blabla/asdwd"
	kasCorrelationClientName                = "gitlab-kas-test"
	agentkToken              api.AgentToken = "123123"
	authSecretKey                           = "blablabla"

	correlationIdHeader         = "X-Request-ID"
	correlationClientNameHeader = "X-GitLab-Client-Name"
)

var (
	_ ClientInterface = &CachingClient{}
)

func TestGetAgentInfo(t *testing.T) {
	ctx, correlationId := ctxWithCorrelation(t)
	response := getAgentInfoResponse{
		ProjectId: 234,
		AgentId:   555,
		AgentName: "agent-x",
		GitalyInfo: gitalyInfo{
			Address: "example.com",
			Token:   "123123",
			Features: map[string]string{
				"a": "b",
			},
		},
		GitalyRepository: gitalyRepository{
			StorageName:   "234",
			RelativePath:  "123",
			GlRepository:  "254634",
			GlProjectPath: "64662",
		},
	}
	r := mux.NewRouter()
	r.Methods(http.MethodGet).Path(agentInfoApiPath).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertGetRequestIsCorrect(t, w, r, correlationId) {
			return
		}

		respondWithJSON(t, w, response)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	c := NewClient(u, []byte(authSecretKey), clientOptionsForTest()...)
	meta := &api.AgentMeta{
		Token: agentkToken,
	}
	agentInfo, err := c.GetAgentInfo(ctx, meta)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectId, agentInfo.ProjectId)
	assert.Equal(t, response.AgentId, agentInfo.Id)
	assert.Equal(t, response.AgentName, agentInfo.Name)

	assertGitalyInfo(t, response.GitalyInfo, agentInfo.GitalyInfo)
	assertGitalyRepository(t, response.GitalyRepository, agentInfo.Repository)
}

func TestGetProjectInfo(t *testing.T) {
	const (
		projectId = "bla/bla"
	)
	ctx, correlationId := ctxWithCorrelation(t)
	response := projectInfoResponse{
		ProjectId: 234,
		GitalyInfo: gitalyInfo{
			Address: "example.com",
			Token:   "123123",
			Features: map[string]string{
				"a": "b",
			},
		},
		GitalyRepository: gitalyRepository{
			StorageName:   "234",
			RelativePath:  "123",
			GlRepository:  "254634",
			GlProjectPath: "64662",
		},
	}
	r := mux.NewRouter()
	r.Methods(http.MethodGet).Path(projectInfoApiPath).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertGetRequestIsCorrect(t, w, r, correlationId) {
			return
		}
		assert.Equal(t, projectId, r.URL.Query().Get(projectIdQueryParam))

		respondWithJSON(t, w, response)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	c := NewClient(u, []byte(authSecretKey), clientOptionsForTest()...)
	meta := &api.AgentMeta{
		Token: agentkToken,
	}
	projectInfo, err := c.GetProjectInfo(ctx, meta, projectId)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectId, projectInfo.ProjectId)
	assertGitalyInfo(t, response.GitalyInfo, projectInfo.GitalyInfo)
	assertGitalyRepository(t, response.GitalyRepository, projectInfo.Repository)
}

func TestSendUsage(t *testing.T) {
	ctx, correlationId := ctxWithCorrelation(t)
	usageData := UsageData{
		GitopsSyncCount: 123,
	}
	r := mux.NewRouter()
	r.Methods(http.MethodPost).Path(usagePingApiPath).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertCommonRequestParams(t, r, correlationId)
		if !assertJWTSignature(t, w, r) {
			return
		}
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		data, err := ioutil.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		var req UsageData
		err = json.Unmarshal(data, &req)
		if !assert.NoError(t, err) {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		assert.Empty(t, cmp.Diff(req, usageData))

		w.WriteHeader(http.StatusNoContent)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	c := NewClient(u, []byte(authSecretKey), clientOptionsForTest()...)
	err = c.SendUsage(ctx, &usageData)
	require.NoError(t, err)
}

func respondWithJSON(t *testing.T, w http.ResponseWriter, response interface{}) {
	data, err := json.Marshal(response)
	if !assert.NoError(t, err) {
		http.Error(w, "json", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func assertGetRequestIsCorrect(t *testing.T, w http.ResponseWriter, r *http.Request, correlationId string) bool {
	assert.Equal(t, "Bearer "+string(agentkToken), r.Header.Get("Authorization"))
	assert.Empty(t, r.Header.Values("Content-Type"))
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
	assertCommonRequestParams(t, r, correlationId)
	return assertJWTSignature(t, w, r)
}

func assertCommonRequestParams(t *testing.T, r *http.Request, correlationId string) {
	assert.Equal(t, kasUserAgent, r.Header.Get("User-Agent"))
	assert.Equal(t, correlationId, r.Header.Get(correlationIdHeader))
	assert.Equal(t, kasCorrelationClientName, r.Header.Get(correlationClientNameHeader))
}

func assertJWTSignature(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	token, err := jwt.Parse(r.Header.Get(jwtRequestHeader), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(authSecretKey), nil
	})
	if !assert.NoError(t, err) {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return false
	}

	if !assert.IsType(t, jwt.MapClaims(nil), token.Claims) {
		http.Error(w, "invalid token claims type", http.StatusBadRequest)
		return false
	}
	claims := token.Claims.(jwt.MapClaims)
	assert.NoError(t, claims.Valid())
	assert.True(t, claims.VerifyIssuer(jwtIssuer, true))
	assert.True(t, claims.VerifyAudience(jwtGitLabAudience, true))
	return true
}

func assertGitalyRepository(t *testing.T, gitalyRepository gitalyRepository, apiGitalyRepository gitalypb.Repository) {
	assert.Equal(t, gitalyRepository.StorageName, apiGitalyRepository.StorageName)
	assert.Equal(t, gitalyRepository.RelativePath, apiGitalyRepository.RelativePath)
	assert.Equal(t, gitalyRepository.GlRepository, apiGitalyRepository.GlRepository)
	assert.Equal(t, gitalyRepository.GlProjectPath, apiGitalyRepository.GlProjectPath)
}

func assertGitalyInfo(t *testing.T, gitalyInfo gitalyInfo, apiGitalyInfo api.GitalyInfo) {
	assert.Equal(t, gitalyInfo.Address, apiGitalyInfo.Address)
	assert.Equal(t, gitalyInfo.Token, apiGitalyInfo.Token)
	assert.Equal(t, gitalyInfo.Features, apiGitalyInfo.Features)
}

func clientOptionsForTest() []ClientOption {
	return []ClientOption{
		WithUserAgent(kasUserAgent),
		WithCorrelationClientName(kasCorrelationClientName),
	}
}

func ctxWithCorrelation(t *testing.T) (context.Context, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	t.Cleanup(cancel)
	correlationId, err := correlation.RandomID()
	require.NoError(t, err)
	ctx = correlation.ContextWithCorrelation(ctx, correlationId)
	return ctx, correlationId
}
