package agent

import (
	"context"

	"github.com/cilium/cilium/api/v1/observer"
	typed_v2 "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/typed/cilium.io/v2"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/cilium_alert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/wait"
)

type module struct {
	log          *zap.Logger
	api          modagent.API
	ciliumClient typed_v2.CiliumV2Interface
}

func (m *module) Run(ctx context.Context, cfg <-chan *agentcfg.AgentConfiguration) error {
	var holder *workerHolder
	defer func() {
		if holder != nil {
			holder.stop()
		}
	}()
	for config := range cfg {
		holder = m.applyNewConfiguration(ctx, holder, config)
	}
	return nil
}

func (m *module) DefaultAndValidateConfiguration(cfg *agentcfg.AgentConfiguration) error {
	return nil
}

func (m *module) Name() string {
	return cilium_alert.ModuleName
}

func (m *module) applyNewConfiguration(ctx context.Context, holder *workerHolder, config *agentcfg.AgentConfiguration) *workerHolder {
	if holder != nil {
		if proto.Equal(config.Cilium, holder.config) {
			// No configuration changes
			return holder
		}
		// Stop to apply new configuration
		holder.stop()
	}
	if config.Cilium == nil {
		// Not configured
		return nil
	}
	// TODO parse the address and check the scheme to see if we need to add WithInsecure()
	clientConn, err := grpc.Dial(config.Cilium.HubbleRelayAddress, grpc.WithInsecure())
	if err != nil {
		m.log.Error("Failed to apply Cilium configuration", zap.Error(err))
		return nil
	}
	newHolder := &workerHolder{
		log:        m.log,
		config:     config.Cilium,
		clientConn: clientConn,
	}
	ctx, newHolder.cancel = context.WithCancel(ctx)
	w := &worker{
		log:            m.log,
		api:            m.api,
		ciliumClient:   m.ciliumClient,
		observerClient: observer.NewObserverClient(clientConn),
		projectId:      config.ProjectId,
	}
	newHolder.wg.StartWithContext(ctx, w.Run)
	return newHolder
}

type workerHolder struct {
	log        *zap.Logger
	wg         wait.Group
	config     *agentcfg.CiliumCF
	cancel     context.CancelFunc
	clientConn *grpc.ClientConn
}

func (h *workerHolder) stop() {
	h.cancel()  // tell worker to stop
	h.wg.Wait() // wait for worker to stop
	// close gRPC connection
	err := h.clientConn.Close()
	if err != nil {
		h.log.Error("Cilium gRPC conn close", zap.Error(err))
	}
}
