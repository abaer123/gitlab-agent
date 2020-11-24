package modagent

import (
	"context"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/agentrpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"google.golang.org/grpc"
)

// API provides the API for the module to use.
type API interface {
	GetObjectsToSynchronize(ctx context.Context, in *agentrpc.ObjectsToSynchronizeRequest, opts ...grpc.CallOption) (agentrpc.Kas_GetObjectsToSynchronizeClient, error)
}

type Factory interface {
	// New creates a new instance of a Module.
	New(API) Module
}

type Module interface {
	// Run starts the module.
	// Run can block until the context is canceled or exit with nil if there is nothing to do.
	Run(context.Context) error
	// DefaultAndValidateConfiguration applies defaults and validates the passed configuration.
	// It is called each time on configuration update before calling SetConfiguration.
	// config is a shared instance, module can mutate only the part of it that it owns.
	DefaultAndValidateConfiguration(config *agentcfg.AgentConfiguration) error
	// SetConfiguration sets configuration to use. It is called each time on configuration update.
	// config is a shared instance, must not be mutated. Module should make a copy if it needs to mutate the object.
	SetConfiguration(config *agentcfg.AgentConfiguration) error
	// Name returns module's name.
	Name() string
}
