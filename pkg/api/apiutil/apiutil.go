package apiutil

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/api"
	"google.golang.org/grpc/credentials"
)

func AgentMetaFromContext(ctx context.Context) (*api.AgentMeta, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}
	return &api.AgentMeta{
		Token:   api.AgentToken(token),
		Version: metautils.ExtractIncoming(ctx).Get(api.MetadataAgentkVersion),
	}, nil
}

func NewTokenCredentials(token string, insecure bool) credentials.PerRPCCredentials {
	return &tokenCredentials{
		token:    token,
		insecure: insecure,
	}
}

type tokenCredentials struct {
	token    string
	insecure bool
}

func (t *tokenCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		api.MetadataAuthorization: "Bearer " + t.token,
	}, nil
}

func (t *tokenCredentials) RequireTransportSecurity() bool {
	return !t.insecure
}
