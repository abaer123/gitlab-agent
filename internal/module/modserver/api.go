package modserver

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/kascfg"
	"gitlab.com/gitlab-org/labkit/errortracking"
	"go.uber.org/zap"
)

// ApplyDefaults is a signature of a public function, exposed by modules to perform defaulting.
// The function should be called ApplyDefaults.
type ApplyDefaults func(*kascfg.ConfigurationFile)

// Config holds configuration for a Module.
type Config struct {
	Log    *zap.Logger
	Config *kascfg.ConfigurationFile
	// Registerer allows to register metrics.
	// Metrics should be registered in Run and unregistered before Run returns.
	Registerer prometheus.Registerer
	// ErrTracker can be used to report errors.
	ErrTracker errortracking.Tracker
	// KasName is a string "gitlab-kas". Can be used as a user agent, server name, service name, etc.
	KasName string
	// Version is gitlab-kas version.
	Version string
	// Commit is gitlab-kas commit.
	Commit string
}

type Factory interface {
	// New creates a new instance of a Module.
	New(*Config) Module
}

type Module interface {
	// Run starts the module.
	// Run can block until the context is canceled or exit with nil if there is nothing to do.
	Run(context.Context) error
	// Name returns module's name.
	Name() string
}
