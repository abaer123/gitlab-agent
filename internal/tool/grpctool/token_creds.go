package grpctool

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/api"
	"google.golang.org/grpc/credentials"
)

const (
	MetadataAuthorization = "authorization"
)

func NewTokenCredentials(token api.AgentToken, insecure bool) credentials.PerRPCCredentials {
	return &tokenCredentials{
		token:    token,
		insecure: insecure,
	}
}

type tokenCredentials struct {
	token    api.AgentToken
	insecure bool
}

func (t *tokenCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		MetadataAuthorization: "Bearer " + string(t.token),
	}, nil
}

func (t *tokenCredentials) RequireTransportSecurity() bool {
	return !t.insecure
}
