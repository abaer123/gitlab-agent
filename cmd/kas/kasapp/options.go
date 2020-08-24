package kasapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"time"

	"github.com/ash2k/stager"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/cmd"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitaly"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/kas"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tools/wstunnel"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/labkit/log"
	"google.golang.org/grpc"
)

const (
	authSecretKeyLength = 32

	defaultReloadConfigurationPeriod = 5 * time.Minute
	defaultMaxMessageSize            = 10 * 1024 * 1024

	defaultAgentInfoCacheTTL      = 5 * time.Minute
	defaultAgentInfoCacheErrorTTL = 1 * time.Minute

	defaultProjectInfoCacheTTL      = 5 * time.Minute
	defaultProjectInfoCacheErrorTTL = 1 * time.Minute
)

type Options struct {
	Configuration *kascfg.ConfigurationFile
}

func (o *Options) Run(ctx context.Context) error {
	// Main logic of kas
	cfg := o.Configuration
	gitLabUrl, err := url.Parse(cfg.GetGitlab().GetAddress())
	if err != nil {
		return err
	}
	// Secret for JWT signing
	decodedAuthSecret, err := o.loadAuthSecret()
	if err != nil {
		return fmt.Errorf("authentication secret: %v", err)
	}

	// gRPC server
	lis, err := net.Listen(cfg.GetListen().GetNetwork(), cfg.GetListen().GetAddress())
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"address": lis.Addr().String(),
		"network": lis.Addr().Network(),
	}).Info("Listening for connections")

	if cfg.GetListen().GetWebsocket() {
		wsWrapper := wstunnel.ListenerWrapper{
			// TODO set timeouts
			ReadLimit: defaultMaxMessageSize,
		}
		lis = wsWrapper.Wrap(lis)
	}

	gitalyClientPool := client.NewPool()
	defer gitalyClientPool.Close() // nolint: errcheck
	gitLabClient := gitlab.NewClient(gitLabUrl, "", decodedAuthSecret, fmt.Sprintf("kas/%s/%s", cmd.Version, cmd.Commit))
	gitLabCachingClient := gitlab.NewCachingClient(gitLabClient, gitlab.CacheOptions{
		CacheTTL:      defaultAgentInfoCacheTTL,
		CacheErrorTTL: defaultAgentInfoCacheErrorTTL,
	}, gitlab.CacheOptions{
		CacheTTL:      defaultProjectInfoCacheTTL,
		CacheErrorTTL: defaultProjectInfoCacheErrorTTL,
	})
	pollPeriod := cfg.GetAgent().GetConfiguration().GetPollPeriod()
	pollPeriodDuration := defaultReloadConfigurationPeriod
	if pollPeriod != nil {
		pollPeriodDuration = pollPeriod.AsDuration()
	}
	srv := &kas.Server{
		Context:                   ctx,
		ReloadConfigurationPeriod: pollPeriodDuration,
		GitalyPool: &gitaly.Pool{
			ClientPool: gitalyClientPool,
		},
		GitLabClient: gitLabCachingClient,
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	agentrpc.RegisterKasServer(grpcServer, srv)

	// Start things up
	st := stager.New()
	defer st.Shutdown()
	stage := st.NextStageWithContext(ctx)
	stage.StartWithContext(func(ctx context.Context) {
		<-ctx.Done() // can be cancelled because Server() failed or because main ctx was cancelled
		grpcServer.GracefulStop()
	})
	return grpcServer.Serve(lis)
}

func (o *Options) loadAuthSecret() ([]byte, error) {
	encodedAuthSecret, err := ioutil.ReadFile(o.Configuration.Gitlab.AuthenticationSecretFile) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("read file: %v", err)
	}
	decodedAuthSecret := make([]byte, authSecretKeyLength)

	n, err := base64.StdEncoding.Decode(decodedAuthSecret, encodedAuthSecret)
	if err != nil {
		return nil, fmt.Errorf("decoding: %v", err)
	}
	if n != authSecretKeyLength {
		return nil, fmt.Errorf("decoding: expecting %d bytes, was %d", authSecretKeyLength, n)
	}
	return decodedAuthSecret, nil
}
