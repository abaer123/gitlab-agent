package server

import (
	"context"

	"cloud.google.com/go/profiler"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/google_profiler"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/kascfg"
	"google.golang.org/api/option"
)

type module struct {
	cfg     *kascfg.GoogleProfilerCF
	service string
	version string
}

func (m *module) Run(ctx context.Context) error {
	if !m.cfg.Enabled {
		return nil
	}
	config := profiler.Config{
		Service:        m.service,
		ServiceVersion: m.version,
		MutexProfiling: true, // like in LabKit
		ProjectID:      m.cfg.ProjectId,
	}
	var opts []option.ClientOption
	if m.cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(m.cfg.CredentialsFile))
	}
	return profiler.Start(config, opts...)
}

func (m *module) Name() string {
	return google_profiler.ModuleName
}
