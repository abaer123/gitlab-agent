package server

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	gapi "gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab/api"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/configuration_project/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/filez"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/gitaly/v14/proto/go/gitalypb"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
)

var (
	_ modserver.Module               = &module{}
	_ modserver.Factory              = &Factory{}
	_ rpc.ConfigurationProjectServer = &server{}
	_ gitaly.FetchVisitor            = (*configVisitor)(nil)
)

func TestConfigVisitor(t *testing.T) {
	tests := []struct {
		path string
		file *rpc.AgentConfigFile
	}{
		{
			path: "asdfdas",
		},
		{
			path: ".gitlab/agents/my-agent/config.yaml",
			file: &rpc.AgentConfigFile{
				Name:      ".gitlab/agents/my-agent/config.yaml",
				AgentName: "my-agent",
			},
		},
		{
			path: ".gitlab/agents/my-agent",
		},
		{
			path: ".gitlab/agents/my-agent/",
		},
		{
			path: ".gitlab/agents/-my-agent-with-invalid-name/config.yaml",
		},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			v := configVisitor{}
			download, _, err := v.Entry(&gitalypb.TreeEntry{
				Path: []byte(tc.path),
			})
			require.NoError(t, err)
			assert.False(t, download)
			var expected []*rpc.AgentConfigFile
			if tc.file != nil {
				expected = []*rpc.AgentConfigFile{tc.file}
			}
			assert.Empty(t, cmp.Diff(v.resp, expected, protocmp.Transform(), cmpopts.EquateEmpty()))
		})
	}
}

func TestAsClient(t *testing.T) {
	kasAddress := os.Getenv("KAS_ADDRESS")
	gitLabAddress := os.Getenv("GITLAB_ADDRESS")
	kasSecretFile := os.Getenv("KAS_SECRET_FILE")
	agentTokenFile := os.Getenv("AGENT_TOKEN_FILE")
	if kasAddress == "" || kasSecretFile == "" || gitLabAddress == "" || agentTokenFile == "" {
		t.SkipNow()
	}
	conn := constructKasConnection(t, kasAddress, kasSecretFile)
	gitLabC := constructGitLabClient(t, gitLabAddress, kasSecretFile)
	agentToken, err := os.ReadFile(agentTokenFile)
	require.NoError(t, err)

	agentInfo, err := gapi.GetAgentInfo(context.TODO(), gitLabC, api.AgentToken(agentToken))
	require.NoError(t, err)

	kasC := rpc.NewConfigurationProjectClient(conn)
	configFiles, err := kasC.ListAgentConfigFiles(context.Background(), &rpc.ListAgentConfigFilesRequest{
		Repository: &modserver.Repository{
			StorageName:                   agentInfo.Repository.StorageName,
			RelativePath:                  agentInfo.Repository.RelativePath,
			GitObjectDirectory:            agentInfo.Repository.GitObjectDirectory,
			GitAlternateObjectDirectories: agentInfo.Repository.GitAlternateObjectDirectories,
			GlRepository:                  agentInfo.Repository.GlRepository,
			GlProjectPath:                 agentInfo.Repository.GlProjectPath,
		},
		GitalyAddress: &modserver.GitalyAddress{
			Address: agentInfo.GitalyInfo.Address,
			Token:   agentInfo.GitalyInfo.Token,
		},
	})
	require.NoError(t, err)
	data, err := protojson.MarshalOptions{
		Multiline: true,
	}.Marshal(configFiles)
	require.NoError(t, err)
	t.Logf("configFiles:\n%s", data)
}

func constructKasConnection(t *testing.T, kasAddress, kasSecretFile string) *grpc.ClientConn {
	jwtSecret, err := filez.LoadBase64Secret(kasSecretFile)
	require.NoError(t, err)
	u, err := url.Parse(kasAddress)
	require.NoError(t, err)
	opts := []grpc.DialOption{
		grpc.WithChainStreamInterceptor(
			grpctool.StreamClientValidatingInterceptor,
		),
		grpc.WithChainUnaryInterceptor(
			grpctool.UnaryClientValidatingInterceptor,
		),
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(&grpctool.JwtCredentials{
			Secret:   jwtSecret,
			Audience: "gitlab-kas",
			Issuer:   "gitlab-kas",
			Insecure: true,
		}),
	}
	var addressToDial string
	switch u.Scheme {
	case "grpc":
		addressToDial = u.Host
	default:
		t.Fatalf("unsupported scheme in GitLab Kubernetes Agent Server address: %q", u.Scheme)
	}
	conn, err := grpc.DialContext(context.Background(), addressToDial, opts...)
	require.NoError(t, err)
	return conn
}

func constructGitLabClient(t *testing.T, gitLabAddress, gitLabSecretFile string) *gitlab.Client {
	gitLabUrl, err := url.Parse(gitLabAddress)
	require.NoError(t, err)
	secret, err := filez.LoadBase64Secret(gitLabSecretFile)
	require.NoError(t, err)
	// Secret for JWT signing
	return gitlab.NewClient(
		gitLabUrl,
		secret,
		gitlab.WithLogger(zaptest.NewLogger(t)),
	)
}
