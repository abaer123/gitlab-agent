package agent

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	cilium_fake "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/fake"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/pkg/agentcfg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestStartsWorkersAccordingToConfiguration(t *testing.T) {
	for caseNum, scenario := range testScenarios() {
		t.Run(fmt.Sprintf("case %d", caseNum), func(t *testing.T) {
			errorEntryCount := int32(0)
			var wg wait.Group
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := setupModule(t, &errorEntryCount)
			cfg := make(chan *agentcfg.AgentConfiguration)
			wg.Start(func() {
				assert.NoError(t, m.Run(ctx, cfg))
			})
			cfg <- scenario.Agentcfg
			time.Sleep(2 * time.Second)
			close(cfg)
			cancel()
			wg.Wait()
			assert.Equal(t, scenario.ErrCount, atomic.LoadInt32(&errorEntryCount))
		})
	}
}

func TestUpdatesWorkersAccordingToConfiguration(t *testing.T) {
	errorEntryCount := int32(0)
	var wg wait.Group
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := setupModule(t, &errorEntryCount)
	cfg := make(chan *agentcfg.AgentConfiguration)
	wg.Start(func() {
		assert.NoError(t, m.Run(ctx, cfg))
	})
	expectedCount := int32(0)
	for _, scenario := range testScenarios() {
		cfg <- scenario.Agentcfg
		time.Sleep(2 * time.Second)
		expectedCount += scenario.ErrCount
	}
	close(cfg)
	cancel()
	wg.Wait()
	assert.Equal(t, expectedCount-1, atomic.LoadInt32(&errorEntryCount)) //-1 because of the same holder is returned when the cilium config does not change
}

func setupModule(t *testing.T, errorEntryCount *int32) *module {
	log := zaptest.NewLogger(t, zaptest.Level(zapcore.DebugLevel))
	log = log.WithOptions(zap.Hooks(logFunction(errorEntryCount)))
	m := &module{
		log:          log,
		api:          mock_modagent.NewMockAPI(gomock.NewController(t)),
		ciliumClient: cilium_fake.NewSimpleClientset(),
		pollConfig:   testhelpers.NewPollConfig(getFlowsPollInterval),
	}
	return m
}

func logFunction(errorEntryCount *int32) func(zapcore.Entry) error {
	return func(ze zapcore.Entry) error {
		if ze.Level == zapcore.ErrorLevel {
			atomic.AddInt32(errorEntryCount, 1)
		}
		return nil
	}
}

type scenario struct {
	Agentcfg *agentcfg.AgentConfiguration
	ErrCount int32
}

func testScenarios() []scenario {
	return []scenario{
		{
			Agentcfg: &agentcfg.AgentConfiguration{},
			ErrCount: 0,
		},
		{
			Agentcfg: &agentcfg.AgentConfiguration{Cilium: nil},
			ErrCount: 0,
		},
		{
			Agentcfg: &agentcfg.AgentConfiguration{Cilium: &agentcfg.CiliumCF{
				HubbleRelayAddress: "127.0.0.1:9000",
			}},
			ErrCount: 1,
		},
		{
			Agentcfg: &agentcfg.AgentConfiguration{Gitops: &agentcfg.GitopsCF{
				ManifestProjects: []*agentcfg.ManifestProjectCF{
					{
						Id: "root/project_1",
					},
				},
			},
				Cilium: &agentcfg.CiliumCF{
					HubbleRelayAddress: "127.0.0.1:9000",
				}},
			ErrCount: 1,
		},
	}
}
