package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/retry"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"gitlab.com/gitlab-org/labkit/correlation"
)

const (
	KasUserAgent                            = "kas/v0.1-blabla/asdwd"
	KasCorrelationClientName                = "gitlab-kas-test"
	AgentkToken              api.AgentToken = "123123"
	AuthSecretKey                           = "blablabla"

	CorrelationIdHeader         = "X-Request-ID"
	CorrelationClientNameHeader = "X-GitLab-Client-Name"

	// Copied from gitlab client package because we don't want to export them

	jwtRequestHeader  = "Gitlab-Kas-Api-Request"
	jwtGitLabAudience = "gitlab"
	jwtIssuer         = "gitlab-kas"

	AgentId   int64 = 123
	ProjectId int64 = 321
)

func RespondWithJSON(t *testing.T, w http.ResponseWriter, response interface{}) {
	data, err := json.Marshal(response)
	if !assert.NoError(t, err) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	assert.NoError(t, err)
}

func AssertRequestMethod(t *testing.T, r *http.Request, method string) {
	assert.Equal(t, method, r.Method)
}

func AssertRequestAccept(t *testing.T, r *http.Request, accept string) {
	assert.Equal(t, accept, r.Header.Get("Accept"))
}

func AssertRequestUserAgent(t *testing.T, r *http.Request, userAgent string) {
	assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
}

func AssertRequestAcceptJson(t *testing.T, r *http.Request) {
	AssertRequestAccept(t, r, "application/json")
}

func AssertRequestContentTypeJson(t *testing.T, r *http.Request) {
	assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
}

func AssertGetJsonRequest(t *testing.T, r *http.Request) {
	AssertRequestMethod(t, r, http.MethodGet)
	AssertRequestAcceptJson(t, r)
}

func AssertAgentToken(t *testing.T, r *http.Request, agentToken api.AgentToken) {
	assert.EqualValues(t, "Bearer "+agentToken, r.Header.Get("Authorization"))
}

func AssertGetJsonRequestIsCorrect(t *testing.T, r *http.Request, correlationId string) {
	AssertRequestAcceptJson(t, r)
	AssertGetRequestIsCorrect(t, r, correlationId)
}

func AssertGetRequestIsCorrect(t *testing.T, r *http.Request, correlationId string) {
	AssertRequestMethod(t, r, http.MethodGet)
	AssertAgentToken(t, r, AgentkToken)
	assert.Empty(t, r.Header.Values("Content-Type"))
	AssertCommonRequestParams(t, r, correlationId)
	AssertJWTSignature(t, r)
}

func AssertCommonRequestParams(t *testing.T, r *http.Request, correlationId string) {
	AssertRequestUserAgent(t, r, KasUserAgent)
	assert.Equal(t, correlationId, r.Header.Get(CorrelationIdHeader))
	assert.Equal(t, KasCorrelationClientName, r.Header.Get(CorrelationClientNameHeader))
}

func AssertJWTSignature(t *testing.T, r *http.Request) {
	_, err := jwt.Parse(r.Header.Get(jwtRequestHeader), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(AuthSecretKey), nil
	}, jwt.WithAudience(jwtGitLabAudience), jwt.WithIssuer(jwtIssuer))
	assert.NoError(t, err)
}

func CtxWithCorrelation(t *testing.T) (context.Context, string) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	correlationId := correlation.SafeRandomID()
	ctx = correlation.ContextWithCorrelation(ctx, correlationId)
	return ctx, correlationId
}

func AgentInfoObj() *api.AgentInfo {
	return &api.AgentInfo{
		Id:        AgentId,
		ProjectId: ProjectId,
		Name:      "agent1",
		GitalyInfo: api.GitalyInfo{
			Address: "127.0.0.1:123123",
			Token:   "abc",
			Features: map[string]string{
				"bla": "true",
			},
		},
		Repository: &gitalypb.Repository{
			StorageName:        "StorageName",
			RelativePath:       "RelativePath",
			GitObjectDirectory: "GitObjectDirectory",
			GlRepository:       "GlRepository",
			GlProjectPath:      "GlProjectPath",
		},
	}
}

func RecvMsg(value interface{}) func(interface{}) {
	return func(msg interface{}) {
		SetValue(msg, value)
	}
}

// SetValue sets target to value.
// target must be a pointer. i.e. *blaProtoMsgType
// value must of the same type as target.
func SetValue(target, value interface{}) {
	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(value).Elem())
}

func NewBackoff() retry.BackoffManagerFactory {
	return retry.NewExponentialBackoffFactory(time.Minute, time.Minute, time.Minute, 2, 1)
}
