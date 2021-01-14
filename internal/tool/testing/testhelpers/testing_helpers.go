package testhelpers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
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

func AssertGetRequestIsCorrect(t *testing.T, w http.ResponseWriter, r *http.Request, correlationId string) bool {
	assert.Equal(t, "Bearer "+string(AgentkToken), r.Header.Get("Authorization"))
	assert.Empty(t, r.Header.Values("Content-Type"))
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
	AssertCommonRequestParams(t, r, correlationId)
	return AssertJWTSignature(t, w, r)
}

func AssertCommonRequestParams(t *testing.T, r *http.Request, correlationId string) {
	assert.Equal(t, KasUserAgent, r.Header.Get("User-Agent"))
	assert.Equal(t, correlationId, r.Header.Get(CorrelationIdHeader))
	assert.Equal(t, KasCorrelationClientName, r.Header.Get(CorrelationClientNameHeader))
}

func AssertJWTSignature(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	_, err := jwt.Parse(r.Header.Get(jwtRequestHeader), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(AuthSecretKey), nil
	}, jwt.WithAudience(jwtGitLabAudience), jwt.WithIssuer(jwtIssuer))
	if !assert.NoError(t, err) {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return true
}

func CtxWithCorrelation(t *testing.T) (context.Context, string) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	correlationId := correlation.SafeRandomID()
	ctx = correlation.ContextWithCorrelation(ctx, correlationId)
	return ctx, correlationId
}

func AssignResult(target, result interface{}) {
	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(result).Elem())
}
