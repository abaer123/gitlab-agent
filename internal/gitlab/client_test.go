package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	kasUserAgent                 = "kas/v0.1-blabla/asdwd"
	agentkToken   api.AgentToken = "123123"
	authSecretKey                = "blablabla"
)

func TestGetAgentInfo(t *testing.T) {
	response := getAgentInfoResponse{
		ProjectID: 234,
		AgentID:   555,
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
		if !assertRequestIsCorrect(t, w, r) {
			return
		}

		respondWithJSON(t, w, response)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	c := NewClient(u, []byte(authSecretKey), kasUserAgent)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	meta := &api.AgentMeta{
		Token: agentkToken,
	}
	agentInfo, err := c.GetAgentInfo(ctx, meta)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectID, agentInfo.ProjectID)
	assert.Equal(t, response.AgentID, agentInfo.ID)
	assert.Equal(t, response.AgentName, agentInfo.Name)

	assertGitalyInfo(t, response.GitalyInfo, agentInfo.GitalyInfo)
	assertGitalyRepository(t, response.GitalyRepository, agentInfo.Repository)
}

func TestGetProjectInfo(t *testing.T) {
	const (
		projectID = "bla/bla"
	)
	response := projectInfoResponse{
		ProjectID: 234,
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
		if !assertRequestIsCorrect(t, w, r) {
			return
		}
		assert.Equal(t, projectID, r.URL.Query().Get(projectIDQueryParam))

		respondWithJSON(t, w, response)
	})
	s := httptest.NewServer(r)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	c := NewClient(u, []byte(authSecretKey), kasUserAgent)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	meta := &api.AgentMeta{
		Token: agentkToken,
	}
	projectInfo, err := c.GetProjectInfo(ctx, meta, projectID)
	require.NoError(t, err)

	assert.Equal(t, response.ProjectID, projectInfo.ProjectID)
	assertGitalyInfo(t, response.GitalyInfo, projectInfo.GitalyInfo)
	assertGitalyRepository(t, response.GitalyRepository, projectInfo.Repository)
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

func assertRequestIsCorrect(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	assert.Equal(t, "Bearer "+string(agentkToken), r.Header.Get("Authorization"))
	assert.Empty(t, r.Header.Values("Content-Type"))
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
	assert.Equal(t, kasUserAgent, r.Header.Get("User-Agent"))
	token, err := jwt.Parse(r.Header.Get(kasRequestHeader), func(token *jwt.Token) (interface{}, error) {
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
	assert.True(t, claims.VerifyIssuer(kasJWTIssuer, true))
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
