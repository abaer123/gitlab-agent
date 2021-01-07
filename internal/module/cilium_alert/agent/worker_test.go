package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/api/v1/observer"
	v2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	"github.com/cilium/cilium/pkg/labels"
	"github.com/cilium/cilium/pkg/policy/api"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/module/modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/internal/tool/testing/mock_modagent"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/pkg/agentcfg"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/testing/protocmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSuccessfulMapping(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker, obsClient, ciliumClient, cnpList, flwClient, mAPI := setupTest(t)
	flow_response := observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
	}}}
	gomock.InOrder(
		obsClient.EXPECT().
			GetFlows(gomock.Any(), gomock.Any()).
			Return(flwClient, nil),
		flwClient.EXPECT().
			Recv().
			Return(&flow_response, nil),
		ciliumClient.EXPECT().
			CiliumNetworkPolicies("ThisNamespace").
			Return(cnpList),
		cnpList.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(&v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "Test"},
					Spec: &api.Rule{
						EndpointSelector: api.NewESFromLabels(labels.NewLabel("thiskey", "", "any")),
						Ingress: []api.IngressRule{api.IngressRule{
							FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
						}},
					},
				}},
			}, nil),
		mAPI.EXPECT().
			MakeGitLabRequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&modagent.GitLabResponse{StatusCode: http.StatusOK}, nil),
		flwClient.EXPECT().
			Recv().
			DoAndReturn(func() (*observer.GetFlowsResponse, error) {
				cancel()
				return nil, errors.New("some error")
			}),
	)
	worker.Run(ctx)
}

func TestNoMatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker, obsClient, ciliumClient, cnpList, flwClient, _ := setupTest(t)
	flow_response := observer.GetFlowsResponse{ResponseTypes: &observer.GetFlowsResponse_Flow{Flow: &flow.Flow{
		Source: &flow.Endpoint{
			Namespace: "ThisNamespace",
			Labels:    []string{"otherkey="},
		},
		TrafficDirection: flow.TrafficDirection_INGRESS,
		Destination:      &flow.Endpoint{Labels: []string{"thiskey="}},
	}}}
	gomock.InOrder(
		obsClient.EXPECT().
			GetFlows(gomock.Any(), gomock.Any()).
			Return(flwClient, nil),
		flwClient.EXPECT().
			Recv().
			Return(&flow_response, nil),
		ciliumClient.EXPECT().
			CiliumNetworkPolicies("ThisNamespace").
			Return(cnpList),
		cnpList.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(&v2.CiliumNetworkPolicyList{
				Items: []v2.CiliumNetworkPolicy{v2.CiliumNetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{Name: "Test"},
					Spec: &api.Rule{
						EndpointSelector: api.NewESFromLabels(labels.NewLabel("notthiskey", "", "any")),
						Ingress: []api.IngressRule{api.IngressRule{
							FromEndpoints: []api.EndpointSelector{api.NewESFromLabels(labels.NewLabel("nootherkey", "", "any"))},
						}},
					},
				}},
			}, nil),
		flwClient.EXPECT().
			Recv().
			DoAndReturn(func() (*observer.GetFlowsResponse, error) {
				cancel()
				return nil, errors.New("some error")
			}),
	)
	worker.Run(ctx)
}

func TestJSON(t *testing.T) {
	p1 := payload{
		Alert: alert{
			Flow: (*flowAlias)(&flow.Flow{
				DropReason: 123,
			}),
			CiliumNetworkPolicy: &v2.CiliumNetworkPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "bla",
					APIVersion: "bla",
				},
			},
		},
	}
	data, err := json.Marshal(p1)
	require.NoError(t, err)

	p2 := payload{}
	err = json.Unmarshal(data, &p2)
	require.NoError(t, err)

	assert.Empty(t, cmp.Diff(p1.Alert.CiliumNetworkPolicy, p2.Alert.CiliumNetworkPolicy))
	assert.Empty(t, cmp.Diff(p1.Alert.Flow, p2.Alert.Flow, protocmp.Transform()))
}

func setupTest(t *testing.T) (*worker, *MockObserverClient, *MockCiliumV2Interface, *MockCiliumNetworkPolicyInterface, *MockObserver_GetFlowsClient, *mock_modagent.MockAPI) {
	ctrl := gomock.NewController(t)
	flwClient := NewMockObserver_GetFlowsClient(ctrl)
	obsClient := NewMockObserverClient(ctrl)
	ciliumClient := NewMockCiliumV2Interface(ctrl)
	cnpList := NewMockCiliumNetworkPolicyInterface(ctrl)
	mAPI := mock_modagent.NewMockAPI(ctrl)
	worker := &worker{
		log:            zaptest.NewLogger(t, zaptest.Level(zapcore.DebugLevel)),
		api:            mAPI,
		config:         &agentcfg.CiliumCF{},
		ciliumClient:   ciliumClient,
		observerClient: obsClient,
	}
	return worker, obsClient, ciliumClient, cnpList, flwClient, mAPI
}
