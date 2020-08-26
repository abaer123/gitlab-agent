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
	"github.com/golang/protobuf/ptypes/duration"
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
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	authSecretLength      = 32
	defaultMaxMessageSize = 10 * 1024 * 1024

	defaultListenNetwork = "tcp"
	defaultListenAddress = "127.0.0.1:0"
	defaultGitLabAddress = "http://localhost:8080"

	defaultAgentConfigurationPollPeriod = 20 * time.Second

	defaultAgentInfoCacheTTL      = 5 * time.Minute
	defaultAgentInfoCacheErrorTTL = 1 * time.Minute

	defaultGitOpsPollPeriod               = 20 * time.Second
	defaultGitOpsProjectInfoCacheTTL      = 5 * time.Minute
	defaultGitOpsProjectInfoCacheErrorTTL = 1 * time.Minute

	// TODO: enable when https://gitlab.com/gitlab-org/gitlab/-/issues/245667 is fixed
	defaultUsageReportingPeriod = 0 * time.Minute
)

type Options struct {
	Configuration *kascfg.ConfigurationFile
}

func (o *Options) Run(ctx context.Context) error {
	// Main logic of kas
	cfg := o.Configuration
	gitLabUrl, err := url.Parse(cfg.Gitlab.Address)
	if err != nil {
		return err
	}
	// Secret for JWT signing
	decodedAuthSecret, err := o.loadAuthSecret()
	if err != nil {
		return fmt.Errorf("authentication secret: %v", err)
	}

	// gRPC server
	lis, err := net.Listen(cfg.Listen.Network, cfg.Listen.Address)
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
	gitLabClient := gitlab.NewClient(gitLabUrl, decodedAuthSecret, fmt.Sprintf("kas/%s/%s", cmd.Version, cmd.Commit))
	gitLabCachingClient := gitlab.NewCachingClient(gitLabClient, gitlab.CacheOptions{
		CacheTTL:      cfg.Agent.InfoCacheTtl.AsDuration(),
		CacheErrorTTL: cfg.Agent.InfoCacheErrorTtl.AsDuration(),
	}, gitlab.CacheOptions{
		CacheTTL:      cfg.Agent.Gitops.ProjectInfoCacheTtl.AsDuration(),
		CacheErrorTTL: cfg.Agent.Gitops.ProjectInfoCacheErrorTtl.AsDuration(),
	})
	srv := &kas.Server{
		Context: ctx,
		GitalyPool: &gitaly.Pool{
			ClientPool: gitalyClientPool,
		},
		GitLabClient:                 gitLabCachingClient,
		AgentConfigurationPollPeriod: cfg.Agent.Configuration.PollPeriod.AsDuration(),
		GitopsPollPeriod:             cfg.Agent.Gitops.PollPeriod.AsDuration(),
		UsageReportingPeriod:         cfg.Metrics.UsageReportingPeriod.AsDuration(),
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	agentrpc.RegisterKasServer(grpcServer, srv)

	// Start things up
	st := stager.New()
	defer st.Shutdown()
	stage := st.NextStage()
	stage.StartWithContext(srv.Run)
	stage = st.NextStageWithContext(ctx)
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
	decodedAuthSecret := make([]byte, authSecretLength)

	n, err := base64.StdEncoding.Decode(decodedAuthSecret, encodedAuthSecret)
	if err != nil {
		return nil, fmt.Errorf("decoding: %v", err)
	}
	if n != authSecretLength {
		return nil, fmt.Errorf("decoding: expecting %d bytes, was %d", authSecretLength, n)
	}
	return decodedAuthSecret, nil
}

func ApplyDefaultsToKasConfigurationFile(cfg *kascfg.ConfigurationFile) {
	if cfg.Listen == nil {
		cfg.Listen = &kascfg.ListenCF{}
	}
	defaultString(&cfg.Listen.Network, defaultListenNetwork)
	defaultString(&cfg.Listen.Address, defaultListenAddress)
	if cfg.Gitlab == nil {
		cfg.Gitlab = &kascfg.GitLabCF{}
	}
	defaultString(&cfg.Gitlab.Address, defaultGitLabAddress)
	if cfg.Agent == nil {
		cfg.Agent = &kascfg.AgentCF{}
	}
	if cfg.Agent.Configuration == nil {
		cfg.Agent.Configuration = &kascfg.AgentConfigurationCF{}
	}
	defaultDuration(&cfg.Agent.Configuration.PollPeriod, defaultAgentConfigurationPollPeriod)
	if cfg.Agent.Gitops == nil {
		cfg.Agent.Gitops = &kascfg.GitopsCF{}
	}
	defaultDuration(&cfg.Agent.Gitops.PollPeriod, defaultGitOpsPollPeriod)
	defaultDuration(&cfg.Agent.Gitops.ProjectInfoCacheTtl, defaultGitOpsProjectInfoCacheTTL)
	defaultDuration(&cfg.Agent.Gitops.ProjectInfoCacheErrorTtl, defaultGitOpsProjectInfoCacheErrorTTL)

	defaultDuration(&cfg.Agent.InfoCacheTtl, defaultAgentInfoCacheTTL)
	defaultDuration(&cfg.Agent.InfoCacheErrorTtl, defaultAgentInfoCacheErrorTTL)

	if cfg.Metrics == nil {
		cfg.Metrics = &kascfg.MetricsCF{}
	}
	defaultDuration(&cfg.Metrics.UsageReportingPeriod, defaultUsageReportingPeriod)
}

func defaultDuration(d **duration.Duration, defaultValue time.Duration) {
	if *d == nil {
		*d = durationpb.New(defaultValue)
	}
}

func defaultString(s *string, defaultValue string) {
	if *s == "" {
		*s = defaultValue
	}
}
